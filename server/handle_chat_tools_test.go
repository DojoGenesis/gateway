package server

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/DojoGenesis/gateway/provider"
)

// toolProvider records every request and answers GenerateCompletion with a
// configurable tool call (or plain text when calls is empty). Its stream
// deliberately carries only text, like every real provider's
// CompletionChunk, so a test can tell whether tool calls survived a stream.
type toolProvider struct {
	calls []provider.ToolCall

	mu         sync.Mutex
	seen       []*provider.CompletionRequest
	streamUsed bool
}

func (p *toolProvider) GetInfo(context.Context) (*provider.ProviderInfo, error) {
	return &provider.ProviderInfo{Name: "tp"}, nil
}
func (p *toolProvider) ListModels(context.Context) ([]provider.ModelInfo, error) {
	return []provider.ModelInfo{{ID: "m1"}}, nil
}
func (p *toolProvider) GenerateCompletion(_ context.Context, req *provider.CompletionRequest) (*provider.CompletionResponse, error) {
	p.mu.Lock()
	p.seen = append(p.seen, req)
	p.mu.Unlock()
	if len(p.calls) > 0 {
		return &provider.CompletionResponse{ToolCalls: p.calls, Usage: provider.Usage{InputTokens: 10, OutputTokens: 5, TotalTokens: 15}}, nil
	}
	return &provider.CompletionResponse{Content: "plain answer"}, nil
}
func (p *toolProvider) GenerateCompletionStream(_ context.Context, req *provider.CompletionRequest) (<-chan *provider.CompletionChunk, error) {
	p.mu.Lock()
	p.seen = append(p.seen, req)
	p.streamUsed = true
	p.mu.Unlock()
	ch := make(chan *provider.CompletionChunk, 2)
	ch <- &provider.CompletionChunk{Delta: "streamed text"}
	ch <- &provider.CompletionChunk{Done: true}
	close(ch)
	return ch, nil
}
func (p *toolProvider) CallTool(context.Context, *provider.ToolCallRequest) (*provider.ToolCallResponse, error) {
	return &provider.ToolCallResponse{Error: "unsupported"}, nil
}
func (p *toolProvider) GenerateEmbedding(context.Context, string) ([]float32, error) { return nil, nil }

func (p *toolProvider) last() *provider.CompletionRequest {
	p.mu.Lock()
	defer p.mu.Unlock()
	if len(p.seen) == 0 {
		return nil
	}
	return p.seen[len(p.seen)-1]
}

func toolServer(p *toolProvider) *Server {
	gin.SetMode(gin.TestMode)
	pm := provider.NewPluginManager("test-plugins")
	pm.RegisterProvider("tp", p)
	return &Server{pluginManager: pm}
}

const weatherTool = `{"type":"function","function":{"name":"get_weather","description":"Weather for a city","parameters":{"type":"object","properties":{"city":{"type":"string"}},"required":["city"]}}}`

// DGS-141: `tools` used to be dropped on decode. The provider must receive
// every tool with its name, description, and parameter schema intact.
func TestTools_RequestToolsReachTheProvider(t *testing.T) {
	p := &toolProvider{}
	s := toolServer(p)
	w := postChat(s, `{"model":"m1","messages":[{"role":"user","content":"weather in Madison?"}],"tools":[`+weatherTool+`],"tool_choice":"required"}`, map[string]string{"X-Route": "direct"})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	req := p.last()
	require.NotNil(t, req)
	require.Len(t, req.Tools, 1, "tools were dropped before reaching the provider")
	assert.Equal(t, "get_weather", req.Tools[0].Name)
	assert.Equal(t, "Weather for a city", req.Tools[0].Description)
	assert.Equal(t, "object", req.Tools[0].Parameters["type"])
	assert.Equal(t, "required", req.ToolChoice)
}

// The model's tool calls must come back in OpenAI shape: arguments as a JSON
// string, type "function", and finish_reason "tool_calls".
func TestTools_ProviderToolCallsComeBackInOpenAIShape(t *testing.T) {
	p := &toolProvider{calls: []provider.ToolCall{{ID: "call_1", Name: "get_weather", Arguments: map[string]interface{}{"city": "Madison"}}}}
	s := toolServer(p)
	w := postChat(s, `{"model":"m1","messages":[{"role":"user","content":"weather?"}],"tools":[`+weatherTool+`]}`, map[string]string{"X-Route": "direct"})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	choice := resp["choices"].([]interface{})[0].(map[string]interface{})
	assert.Equal(t, "tool_calls", choice["finish_reason"])
	msg := choice["message"].(map[string]interface{})
	calls, ok := msg["tool_calls"].([]interface{})
	require.True(t, ok, "response message has no tool_calls: %s", w.Body.String())
	require.Len(t, calls, 1)
	call := calls[0].(map[string]interface{})
	assert.Equal(t, "call_1", call["id"])
	assert.Equal(t, "function", call["type"])
	fn := call["function"].(map[string]interface{})
	assert.Equal(t, "get_weather", fn["name"])
	args, isString := fn["arguments"].(string)
	require.True(t, isString, "arguments must be a JSON string on the OpenAI wire")
	assert.JSONEq(t, `{"city":"Madison"}`, args)
}

// A multi-turn tool conversation: the assistant's tool_calls and the tool
// result (role "tool" + tool_call_id) must reach the provider, or the model
// never sees the result it asked for.
func TestTools_ToolCallHistoryReachesTheProvider(t *testing.T) {
	p := &toolProvider{}
	s := toolServer(p)
	body := `{"model":"m1","tools":[` + weatherTool + `],"messages":[
		{"role":"user","content":"weather?"},
		{"role":"assistant","content":null,"tool_calls":[{"id":"call_1","type":"function","function":{"name":"get_weather","arguments":"{\"city\":\"Madison\"}"}}]},
		{"role":"tool","tool_call_id":"call_1","content":"{\"temp_f\":61}"}]}`
	w := postChat(s, body, map[string]string{"X-Route": "direct"})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	msgs := p.last().Messages
	require.Len(t, msgs, 3)
	require.Len(t, msgs[1].ToolCalls, 1, "the assistant's tool_calls were dropped")
	assert.Equal(t, "call_1", msgs[1].ToolCalls[0].ID)
	assert.Equal(t, "get_weather", msgs[1].ToolCalls[0].Name)
	assert.Equal(t, "Madison", msgs[1].ToolCalls[0].Arguments["city"])
	assert.Equal(t, "tool", msgs[2].Role)
	assert.Equal(t, "call_1", msgs[2].ToolCallID, "the tool result lost its tool_call_id")
	assert.Equal(t, `{"temp_f":61}`, msgs[2].Content)
}

// A follow-up turn often has no NEW user message after the tool result. The
// request still has a user message earlier in the history, so it is valid.
func TestTools_AToolResultTurnIsNotRefusedForLackingATrailingUserMessage(t *testing.T) {
	p := &toolProvider{}
	s := toolServer(p)
	body := `{"model":"m1","tools":[` + weatherTool + `],"messages":[
		{"role":"user","content":"weather?"},
		{"role":"assistant","tool_calls":[{"id":"c","type":"function","function":{"name":"get_weather","arguments":"{}"}}]},
		{"role":"tool","tool_call_id":"c","content":"sunny"}]}`
	w := postChat(s, body, map[string]string{"X-Route": "direct"})
	assert.Equal(t, http.StatusOK, w.Code, w.Body.String())
}

// Malformed tool-call arguments are a client error, never a silent drop.
func TestTools_MalformedToolCallArgumentsAre400(t *testing.T) {
	p := &toolProvider{}
	s := toolServer(p)
	body := `{"model":"m1","messages":[
		{"role":"user","content":"x"},
		{"role":"assistant","tool_calls":[{"id":"c","type":"function","function":{"name":"f","arguments":"{not json"}}]}]}`
	w := postChat(s, body, map[string]string{"X-Route": "direct"})
	assert.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
	assert.Nil(t, p.last(), "the provider must not be called")
}

// tool_choice naming one function has no equivalent in the provider layer.
// Quietly treating it as "auto" would be the silent drop DGS-141 is about.
func TestTools_ObjectToolChoiceIsRefusedByName(t *testing.T) {
	p := &toolProvider{}
	s := toolServer(p)
	w := postChat(s, `{"model":"m1","messages":[{"role":"user","content":"x"}],"tools":[`+weatherTool+`],"tool_choice":{"type":"function","function":{"name":"get_weather"}}}`, map[string]string{"X-Route": "direct"})
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "tool_choice")
	assert.Nil(t, p.last())
}

// Provider stream chunks carry text only, so a streamed request with tools is
// served by one non-streaming call and emitted as spec-valid SSE: the tool
// calls arrive as a delta and the finish reason is "tool_calls".
func TestTools_StreamWithToolsDeliversToolCallsAsSSE(t *testing.T) {
	p := &toolProvider{calls: []provider.ToolCall{{ID: "call_1", Name: "get_weather", Arguments: map[string]interface{}{"city": "Madison"}}}}
	s := toolServer(p)
	w := postChat(s, `{"model":"m1","stream":true,"messages":[{"role":"user","content":"weather?"}],"tools":[`+weatherTool+`]}`, map[string]string{"X-Route": "direct"})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.False(t, p.streamUsed, "the text-only provider stream would lose the tool calls")

	body := w.Body.String()
	require.True(t, strings.HasSuffix(strings.TrimSpace(body), "data: [DONE]"), body)
	var sawCall, sawFinish bool
	for _, line := range strings.Split(body, "\n") {
		if !strings.HasPrefix(line, "data: {") {
			continue
		}
		var chunk map[string]interface{}
		require.NoError(t, json.Unmarshal([]byte(strings.TrimPrefix(line, "data: ")), &chunk))
		choice := chunk["choices"].([]interface{})[0].(map[string]interface{})
		if fr, ok := choice["finish_reason"].(string); ok && fr == "tool_calls" {
			sawFinish = true
		}
		if delta, ok := choice["delta"].(map[string]interface{}); ok {
			if calls, ok := delta["tool_calls"].([]interface{}); ok && len(calls) == 1 {
				call := calls[0].(map[string]interface{})
				assert.EqualValues(t, 0, call["index"])
				fn := call["function"].(map[string]interface{})
				assert.Equal(t, "get_weather", fn["name"])
				assert.JSONEq(t, `{"city":"Madison"}`, fn["arguments"].(string))
				sawCall = true
			}
		}
	}
	assert.True(t, sawCall, "no tool_calls delta in the stream:\n%s", body)
	assert.True(t, sawFinish, "no finish_reason tool_calls in the stream:\n%s", body)
}

// Requests without tools are untouched: a stream still uses the provider's
// real stream, and a plain answer still finishes with "stop".
func TestTools_RequestsWithoutToolsAreUnchanged(t *testing.T) {
	p := &toolProvider{}
	s := toolServer(p)
	w := postChat(s, `{"model":"m1","stream":true,"messages":[{"role":"user","content":"hi"}]}`, map[string]string{"X-Route": "direct"})
	require.Equal(t, http.StatusOK, w.Code)
	assert.True(t, p.streamUsed, "a tool-less stream must still use the provider's stream")
	assert.Empty(t, p.last().Tools)

	w = postChat(s, `{"model":"m1","messages":[{"role":"user","content":"hi"}]}`, map[string]string{"X-Route": "direct"})
	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"finish_reason":"stop"`)
	assert.NotContains(t, w.Body.String(), "tool_calls")
}
