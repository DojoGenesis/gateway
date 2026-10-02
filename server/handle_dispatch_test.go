package server

// DGS-149 /dispatch tests. They drive the REAL setupMiddleware + setupRoutes
// wiring (like router_auth_test.go) so the auth gate and the two route
// registrations are exercised, not a hand-built router.

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/DojoGenesis/gateway/provider"
)

const testKataDir = "testdata/kata"

type dispatchFakeProvider struct {
	resp *provider.CompletionResponse
	err  error

	mu   sync.Mutex
	seen []*provider.CompletionRequest
}

func (p *dispatchFakeProvider) GetInfo(context.Context) (*provider.ProviderInfo, error) {
	return &provider.ProviderInfo{Name: "fake"}, nil
}
func (p *dispatchFakeProvider) ListModels(context.Context) ([]provider.ModelInfo, error) {
	return []provider.ModelInfo{{ID: "m1"}}, nil
}
func (p *dispatchFakeProvider) GenerateCompletion(_ context.Context, req *provider.CompletionRequest) (*provider.CompletionResponse, error) {
	p.mu.Lock()
	p.seen = append(p.seen, req)
	p.mu.Unlock()
	return p.resp, p.err
}
func (p *dispatchFakeProvider) GenerateCompletionStream(context.Context, *provider.CompletionRequest) (<-chan *provider.CompletionChunk, error) {
	return nil, errors.New("not used")
}
func (p *dispatchFakeProvider) CallTool(context.Context, *provider.ToolCallRequest) (*provider.ToolCallResponse, error) {
	return nil, errors.New("not used")
}
func (p *dispatchFakeProvider) GenerateEmbedding(context.Context, string) ([]float32, error) {
	return nil, nil
}
func (p *dispatchFakeProvider) calls() []*provider.CompletionRequest {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]*provider.CompletionRequest(nil), p.seen...)
}

func emit(args map[string]interface{}) *provider.CompletionResponse {
	return &provider.CompletionResponse{
		Model:     "m1",
		ToolCalls: []provider.ToolCall{{ID: "c1", Name: emitResultTool, Arguments: args}},
		Usage:     provider.Usage{InputTokens: 11, OutputTokens: 7, TotalTokens: 18},
	}
}

func newDispatchTestServer(t *testing.T, p provider.ModelProvider) *Server {
	t.Helper()
	gin.SetMode(gin.TestMode)
	pm := provider.NewPluginManager("test-plugins")
	if p != nil {
		pm.RegisterProvider("fake", p)
	}
	s := &Server{
		router: gin.New(),
		cfg: &ServerConfig{
			Port:           "7340",
			Environment:    "test",
			AuthMode:       "api_key",
			AllowedOrigins: []string{"*"},
		},
		pluginManager:  pm,
		agents:         map[string]*AgentRuntime{},
		orchestrations: NewOrchestrationStore(),
		dispatch:       loadDispatchRegistry(testKataDir, "", 2*time.Second),
	}
	s.setupMiddleware()
	s.setupRoutes()
	return s
}

func tokenFor(t *testing.T, sub string) string {
	t.Helper()
	tok, err := issueToken(sub, "user", time.Hour)
	require.NoError(t, err)
	return tok
}

func postDispatch(t *testing.T, s *Server, path, token, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Request-ID", "req-123")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)
	return w
}

const okBody = `{"skill":"kata-test-echo","contract":"kata-ai/v1","mode":"focused","session":{"app":"kata"},"input":{"goal":"learn to juggle"}}`

func TestDispatch_RegistryLoad(t *testing.T) {
	reg := loadDispatchRegistry(testKataDir, "", time.Second)
	require.Contains(t, reg.structured, "kata-test-echo")
	assert.NotContains(t, reg.structured, "kata-plain", "non-structured skill must not be dispatchable")
	assert.NotContains(t, reg.structured, "kata-broken", "malformed op must be skipped, not crash")
	op := reg.structured["kata-test-echo"]
	assert.Equal(t, "1.2.3", op.Version)
	assert.Equal(t, "kata-ai/v1", op.Contract)
	assert.Equal(t, "You are a test op. Restate the goal as one step.", op.System)
	_, hasSchema := op.toolParams["$schema"]
	_, hasID := op.toolParams["$id"]
	assert.False(t, hasSchema || hasID, "$schema/$id must be stripped from tool parameters")
	assert.Contains(t, op.toolParams, "$defs", "internal $defs must survive")

	empty := loadDispatchRegistry(t.TempDir()+"/nope", "", time.Second)
	assert.Empty(t, empty.structured, "missing plugin dir → zero structured ops")
}

func TestDispatch_Success(t *testing.T) {
	for _, path := range []string{"/dispatch", "/v1/dispatch"} {
		t.Run(path, func(t *testing.T) {
			p := &dispatchFakeProvider{resp: emit(map[string]interface{}{"steps": []interface{}{map[string]interface{}{"title": "Toss one ball", "minutes": float64(5)}}})}
			s := newDispatchTestServer(t, p)
			w := postDispatch(t, s, path, tokenFor(t, "u1"), okBody)
			require.Equal(t, http.StatusOK, w.Code, w.Body.String())
			assert.Equal(t, "req-123", w.Header().Get("X-Request-ID"))
			assert.Equal(t, "none", w.Header().Get("X-Dojo-Injected"))

			var resp map[string]interface{}
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
			assert.JSONEq(t, `{"steps":[{"title":"Toss one ball","minutes":5}]}`, mustJSON(t, resp["output"]))
			prov := resp["provenance"].(map[string]interface{})
			assert.Equal(t, "kata-test-echo", prov["skill"])
			assert.Equal(t, "1.2.3", prov["skill_version"])
			assert.Equal(t, "kata-ai/v1", prov["contract"])
			assert.Equal(t, "m1", prov["model"])
			assert.Equal(t, "req-123", prov["request_id"])
			ts, err := time.Parse(time.RFC3339, prov["generated_at"].(string))
			require.NoError(t, err)
			assert.Equal(t, time.UTC, ts.Location())
			assert.JSONEq(t, `{"prompt_tokens":11,"completion_tokens":7,"total_tokens":18}`, mustJSON(t, resp["usage"]))
		})
	}
}

// No base system prompt, no RAG: the provider sees exactly the op's system
// prompt and the input JSON, with emit_result forced.
func TestDispatch_NoInjection(t *testing.T) {
	t.Setenv("SYSTEM_PROMPT", "INJECTED BASE PROMPT")
	p := &dispatchFakeProvider{resp: emit(map[string]interface{}{"steps": []interface{}{map[string]interface{}{"title": "x"}}})}
	s := newDispatchTestServer(t, p)
	w := postDispatch(t, s, "/dispatch", tokenFor(t, "u1"), okBody)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	calls := p.calls()
	require.Len(t, calls, 1)
	req := calls[0]
	require.Len(t, req.Messages, 2, "only system + user may reach the provider")
	assert.Equal(t, "system", req.Messages[0].Role)
	assert.Equal(t, "You are a test op. Restate the goal as one step.", req.Messages[0].Content)
	assert.Equal(t, "user", req.Messages[1].Role)
	assert.JSONEq(t, `{"goal":"learn to juggle"}`, req.Messages[1].Content)
	assert.Equal(t, "m1", req.Model)
	assert.InDelta(t, 0.2, req.Temperature, 1e-9)
	assert.Equal(t, 512, req.MaxTokens)
	assert.Equal(t, "required", req.ToolChoice)
	require.Len(t, req.Tools, 1)
	assert.Equal(t, emitResultTool, req.Tools[0].Name)
	assert.Equal(t, "Return the result for this request.", req.Tools[0].Description)
	assert.Equal(t, "object", req.Tools[0].Parameters["type"])

	// Control: the same server and env DO inject on /v1/chat/completions, so
	// the assertion above is not vacuous.
	creq := httptest.NewRequest(http.MethodPost, "/v1/chat/completions",
		strings.NewReader(`{"model":"m1","messages":[{"role":"user","content":"hi"}]}`))
	creq.Header.Set("Content-Type", "application/json")
	creq.Header.Set("Authorization", "Bearer "+tokenFor(t, "u1"))
	cw := httptest.NewRecorder()
	s.router.ServeHTTP(cw, creq)
	require.Equal(t, http.StatusOK, cw.Code, cw.Body.String())
	calls = p.calls()
	require.Len(t, calls, 2)
	assert.Equal(t, "INJECTED BASE PROMPT", calls[1].Messages[0].Content, "control: chat path injects the base prompt")
}

func TestDispatch_Errors(t *testing.T) {
	good := emit(map[string]interface{}{"steps": []interface{}{map[string]interface{}{"title": "ok"}}})
	cases := []struct {
		name     string
		resp     *provider.CompletionResponse
		provErr  error
		body     string
		sub      string
		status   int
		code     string
		noRetry  bool
		noCall   bool
		provider bool
	}{
		{name: "malformed body", resp: good, body: `{"skill":`, status: 400, code: "invalid_request", noRetry: true, noCall: true},
		{name: "missing skill", resp: good, body: `{"input":{}}`, status: 400, code: "invalid_request", noRetry: true, noCall: true},
		{name: "missing input", resp: good, body: `{"skill":"kata-test-echo","contract":"kata-ai/v1"}`, status: 400, code: "invalid_request", noRetry: true, noCall: true},
		{name: "input not object", resp: good, body: `{"skill":"kata-test-echo","contract":"kata-ai/v1","input":[1]}`, status: 400, code: "invalid_request", noRetry: true, noCall: true},
		{name: "bad mode", resp: good, body: `{"skill":"kata-test-echo","contract":"kata-ai/v1","mode":"wild","input":{"goal":"g"}}`, status: 400, code: "invalid_request", noRetry: true, noCall: true},
		{name: "input fails schema", resp: good, body: `{"skill":"kata-test-echo","contract":"kata-ai/v1","input":{"goal":""}}`, status: 400, code: "invalid_input", noRetry: true, noCall: true},
		{name: "input extra field", resp: good, body: `{"skill":"kata-test-echo","contract":"kata-ai/v1","input":{"goal":"g","x":1}}`, status: 400, code: "invalid_input", noRetry: true, noCall: true},
		{name: "subject mismatch in input", resp: good, body: `{"skill":"kata-test-echo","contract":"kata-ai/v1","input":{"goal":"g","user_id":"someone-else"}}`, status: 400, code: "subject_mismatch", noRetry: true, noCall: true},
		{name: "subject mismatch in body", resp: good, body: `{"skill":"kata-test-echo","contract":"kata-ai/v1","user_id":"someone-else","input":{"goal":"g"}}`, status: 400, code: "subject_mismatch", noRetry: true, noCall: true},
		{name: "subject mismatch non-string", resp: good, body: `{"skill":"kata-test-echo","contract":"kata-ai/v1","input":{"goal":"g","user_id":7}}`, status: 400, code: "subject_mismatch", noRetry: true, noCall: true},
		{name: "unknown skill", resp: good, body: `{"skill":"nope","contract":"kata-ai/v1","input":{}}`, status: 404, code: "unknown_skill", noRetry: true, noCall: true},
		{name: "non-structured skill is unknown", resp: good, body: `{"skill":"kata-plain","input":{}}`, status: 404, code: "unknown_skill", noRetry: true, noCall: true},
		{name: "contract missing", resp: good, body: `{"skill":"kata-test-echo","input":{"goal":"g"}}`, status: 422, code: "unsupported_contract", noRetry: true, noCall: true},
		{name: "contract wrong", resp: good, body: `{"skill":"kata-test-echo","contract":"kata-ai/v2","input":{"goal":"g"}}`, status: 422, code: "unsupported_contract", noRetry: true, noCall: true},
		{name: "blocked go op memory-write", resp: good, body: `{"skill":"kata-memory-write","input":{"user_id":"u1","roll":{}}}`, status: 501, code: "not_implemented", noRetry: true, noCall: true},
		{name: "blocked go op account-delete", resp: good, body: `{"skill":"kata-account-delete","input":{"user_id":"u1"}}`, status: 501, code: "not_implemented", noRetry: true, noCall: true},
		{name: "go op still checks subject", resp: good, body: `{"skill":"kata-account-delete","input":{"user_id":"victim"}}`, status: 400, code: "subject_mismatch", noRetry: true, noCall: true},
		{name: "no tool call", resp: &provider.CompletionResponse{Content: "here you go"}, body: okBody, status: 502, code: "invalid_model_output", noRetry: true},
		{name: "wrong tool name", resp: &provider.CompletionResponse{ToolCalls: []provider.ToolCall{{Name: "other", Arguments: map[string]interface{}{}}}}, body: okBody, status: 502, code: "invalid_model_output", noRetry: true},
		{name: "unparsable args (nil map)", resp: &provider.CompletionResponse{ToolCalls: []provider.ToolCall{{Name: emitResultTool}}}, body: okBody, status: 502, code: "invalid_model_output", noRetry: true},
		{name: "schema-invalid output", resp: emit(map[string]interface{}{"steps": []interface{}{}}), body: okBody, status: 502, code: "invalid_model_output", noRetry: true},
		{name: "lint-tripping output", resp: emit(map[string]interface{}{"steps": []interface{}{map[string]interface{}{"title": "Don't be Lazy"}}}), body: okBody, status: 502, code: "invalid_model_output", noRetry: true},
		{name: "provider error", provErr: errors.New("upstream 529"), body: okBody, status: 503, code: "provider_unavailable", noRetry: false},
		{name: "provider timeout", provErr: context.DeadlineExceeded, body: okBody, status: 503, code: "provider_unavailable", noRetry: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := &dispatchFakeProvider{resp: tc.resp, err: tc.provErr}
			s := newDispatchTestServer(t, p)
			sub := tc.sub
			if sub == "" {
				sub = "u1"
			}
			w := postDispatch(t, s, "/dispatch", tokenFor(t, sub), tc.body)
			require.Equal(t, tc.status, w.Code, w.Body.String())
			var resp map[string]interface{}
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
			assert.Equal(t, tc.code, resp["error"], "flat error code")
			assert.NotEmpty(t, resp["message"])
			if tc.noRetry {
				assert.Equal(t, "true", w.Header().Get("X-No-Retry"))
			} else {
				assert.Empty(t, w.Header().Get("X-No-Retry"), "503 must stay retriable")
			}
			if tc.noCall {
				assert.Empty(t, p.calls(), "provider must not be called")
			}
		})
	}
}

func TestDispatch_NoProviderIs503(t *testing.T) {
	s := newDispatchTestServer(t, nil)
	w := postDispatch(t, s, "/dispatch", tokenFor(t, "u1"), okBody)
	require.Equal(t, http.StatusServiceUnavailable, w.Code, w.Body.String())
	assert.Empty(t, w.Header().Get("X-No-Retry"))
}

func TestDispatch_UnauthenticatedIs401(t *testing.T) {
	p := &dispatchFakeProvider{resp: emit(map[string]interface{}{"steps": []interface{}{map[string]interface{}{"title": "x"}}})}
	s := newDispatchTestServer(t, p)
	for _, path := range []string{"/dispatch", "/v1/dispatch"} {
		for _, tok := range []string{"", "not-a-jwt"} {
			w := postDispatch(t, s, path, tok, okBody)
			assert.Equal(t, http.StatusUnauthorized, w.Code, "%s token=%q: %s", path, tok, w.Body.String())
		}
	}
	assert.Empty(t, p.calls(), "an unauthenticated request must never reach the provider")
}

func TestLintViolations_Recursive(t *testing.T) {
	rules, err := parseLintRules([]byte(`{"rules":[{"id":"r1","pattern":"bad","flags":"i"},{"id":"r2","pattern":"Case"}]}`))
	require.NoError(t, err)
	v := map[string]interface{}{"a": []interface{}{"fine", map[string]interface{}{"b": "BAD thing"}}, "c": "case", "n": 3}
	hits := lintViolations(rules, v)
	assert.Equal(t, []string{"r1 at /a/1/b"}, hits, "i flag folds case; no flag is case-sensitive")
}

func mustJSON(t *testing.T, v interface{}) string {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)
	return string(b)
}
