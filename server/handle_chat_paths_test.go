package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/DojoGenesis/gateway/provider"
)

// capturingProvider records the CompletionRequest each call receives, so a
// test can assert on exactly what the handler sent the model.
type capturingProvider struct {
	name   string
	models []provider.ModelInfo

	mu   sync.Mutex
	seen []*provider.CompletionRequest
}

func (p *capturingProvider) GetInfo(context.Context) (*provider.ProviderInfo, error) {
	return &provider.ProviderInfo{Name: p.name}, nil
}
func (p *capturingProvider) ListModels(context.Context) ([]provider.ModelInfo, error) {
	return p.models, nil
}
func (p *capturingProvider) record(req *provider.CompletionRequest) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.seen = append(p.seen, req)
}
func (p *capturingProvider) last() *provider.CompletionRequest {
	p.mu.Lock()
	defer p.mu.Unlock()
	if len(p.seen) == 0 {
		return nil
	}
	return p.seen[len(p.seen)-1]
}
func (p *capturingProvider) GenerateCompletion(_ context.Context, req *provider.CompletionRequest) (*provider.CompletionResponse, error) {
	p.record(req)
	return &provider.CompletionResponse{Content: "ok from " + p.name}, nil
}
func (p *capturingProvider) GenerateCompletionStream(_ context.Context, req *provider.CompletionRequest) (<-chan *provider.CompletionChunk, error) {
	p.record(req)
	ch := make(chan *provider.CompletionChunk, 2)
	ch <- &provider.CompletionChunk{Delta: "ok from " + p.name}
	ch <- &provider.CompletionChunk{Done: true}
	close(ch)
	return ch, nil
}
func (p *capturingProvider) CallTool(context.Context, *provider.ToolCallRequest) (*provider.ToolCallResponse, error) {
	return &provider.ToolCallResponse{Error: "unsupported"}, nil
}
func (p *capturingProvider) GenerateEmbedding(context.Context, string) ([]float32, error) {
	return nil, nil
}

func chatServer(provs ...*capturingProvider) *Server {
	gin.SetMode(gin.TestMode)
	pm := provider.NewPluginManager("test-plugins")
	for _, p := range provs {
		pm.RegisterProvider(p.name, p)
	}
	return &Server{pluginManager: pm}
}

func postChat(s *Server, body string, headers map[string]string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		c.Request.Header.Set(k, v)
	}
	s.handleChatCompletions(c)
	return w
}

func roles(req *provider.CompletionRequest) string {
	var r []string
	for _, m := range req.Messages {
		r = append(r, m.Role)
	}
	return strings.Join(r, ",")
}

// Both paths — stream and non-stream — must build the same model input:
// the base system prompt is prepended only when the caller sent no system
// message and did not ask for X-Route: direct.
func TestChatPaths_InjectionRulesAreIdenticalForStreamAndNonStream(t *testing.T) {
	t.Setenv("SYSTEM_PROMPT", "BASE PROMPT")
	t.Setenv("SYSTEM_PROMPT_FILE", "")

	for _, stream := range []string{"false", "true"} {
		for _, tc := range []struct {
			name      string
			messages  string
			headers   map[string]string
			wantRoles string
			wantFirst string
		}{
			{"no system message: base prompt injected", `[{"role":"user","content":"hi"}]`, nil, "system,user", "BASE PROMPT"},
			{"caller system message: nothing injected", `[{"role":"system","content":"mine"},{"role":"user","content":"hi"}]`, nil, "system,user", "mine"},
			{"X-Route direct: nothing injected", `[{"role":"user","content":"hi"}]`, map[string]string{"X-Route": "direct"}, "user", "hi"},
		} {
			t.Run("stream="+stream+"/"+tc.name, func(t *testing.T) {
				p := &capturingProvider{name: "p1", models: []provider.ModelInfo{{ID: "m1"}}}
				s := chatServer(p)
				w := postChat(s, `{"model":"m1","stream":`+stream+`,"messages":`+tc.messages+`}`, tc.headers)
				if w.Code != http.StatusOK {
					t.Fatalf("status %d: %s", w.Code, w.Body.String())
				}
				req := p.last()
				if req == nil {
					t.Fatal("provider was never called")
				}
				if got := roles(req); got != tc.wantRoles {
					t.Errorf("roles = %q, want %q", got, tc.wantRoles)
				}
				if req.Messages[0].Content != tc.wantFirst {
					t.Errorf("first message = %q, want %q", req.Messages[0].Content, tc.wantFirst)
				}
				if req.Stream != (stream == "true") {
					t.Errorf("Stream = %v, want %s", req.Stream, stream)
				}
				if req.Temperature != 0.7 || req.MaxTokens != 4096 {
					t.Errorf("defaults: temperature %v max_tokens %d, want 0.7 / 4096", req.Temperature, req.MaxTokens)
				}
			})
		}
	}
}

// A request with no user message has nothing to answer. The non-stream path
// always refused it with 400; the stream path used to send it to the model
// anyway. One rule for both.
func TestChatPaths_NoUserMessageIsRefusedOnBothPaths(t *testing.T) {
	for _, stream := range []string{"false", "true"} {
		p := &capturingProvider{name: "p1", models: []provider.ModelInfo{{ID: "m1"}}}
		s := chatServer(p)
		w := postChat(s, `{"model":"m1","stream":`+stream+`,"messages":[{"role":"system","content":"only system"}]}`, nil)
		if w.Code != http.StatusBadRequest {
			t.Errorf("stream=%s: status %d, want 400", stream, w.Code)
		}
		if p.last() != nil {
			t.Errorf("stream=%s: provider was called for a request with no user message", stream)
		}
	}
}
