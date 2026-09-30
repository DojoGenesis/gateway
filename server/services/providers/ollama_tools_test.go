package providers

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/DojoGenesis/gateway/provider"
)

// fakeOllama answers /api/show with the given capabilities and /api/chat with
// a tool call (when replyWithCall) or plain text, recording each chat body.
type fakeOllama struct {
	capabilities  []string // nil: /api/show omits the field (older Ollama)
	replyWithCall bool

	mu     sync.Mutex
	bodies []map[string]interface{}
	shows  int
}

func (f *fakeOllama) server(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/show":
			f.mu.Lock()
			f.shows++
			f.mu.Unlock()
			out := map[string]interface{}{"details": map[string]interface{}{}}
			if f.capabilities != nil {
				out["capabilities"] = f.capabilities
			}
			_ = json.NewEncoder(w).Encode(out)
		case "/api/chat":
			raw, _ := io.ReadAll(r.Body)
			var body map[string]interface{}
			_ = json.Unmarshal(raw, &body)
			f.mu.Lock()
			f.bodies = append(f.bodies, body)
			f.mu.Unlock()
			msg := map[string]interface{}{"role": "assistant", "content": "plain"}
			if f.replyWithCall {
				// The shape Ollama 0.32 returns: arguments are an OBJECT.
				msg = map[string]interface{}{"role": "assistant", "content": "",
					"tool_calls": []interface{}{map[string]interface{}{"id": "call_x1",
						"function": map[string]interface{}{"index": 0, "name": "get_weather", "arguments": map[string]interface{}{"city": "Madison"}}}}}
			}
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"model": body["model"], "message": msg, "done": true})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

func (f *fakeOllama) lastBody() map[string]interface{} {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.bodies) == 0 {
		return nil
	}
	return f.bodies[len(f.bodies)-1]
}

func ollamaFor(srv *httptest.Server) *OllamaProvider {
	return &OllamaProvider{BaseProvider: BaseProvider{Name: "ollama", BaseURL: srv.URL, Client: srv.Client()}}
}

var weather = provider.Tool{Name: "get_weather", Description: "Weather for a city",
	Parameters: map[string]interface{}{"type": "object", "properties": map[string]interface{}{"city": map[string]interface{}{"type": "string"}}}}

// DGS-141, one layer down: the "native" path never sent tools to Ollama and
// never read tool_calls back, so every tool-capable model silently lost them.
func TestOllama_NativeToolsAreSentAndToolCallsParsed(t *testing.T) {
	f := &fakeOllama{capabilities: []string{"completion", "tools"}, replyWithCall: true}
	p := ollamaFor(f.server(t))

	resp, err := p.GenerateCompletion(context.Background(), &provider.CompletionRequest{
		Model: "qwen2.5:14b", Messages: []provider.Message{{Role: "user", Content: "weather?"}}, Tools: []provider.Tool{weather},
	})
	require.NoError(t, err)

	tools, ok := f.lastBody()["tools"].([]interface{})
	require.True(t, ok, "no tools in the request Ollama received: %v", f.lastBody())
	fn := tools[0].(map[string]interface{})["function"].(map[string]interface{})
	assert.Equal(t, "get_weather", fn["name"])
	assert.Equal(t, "object", fn["parameters"].(map[string]interface{})["type"])

	require.Len(t, resp.ToolCalls, 1, "Ollama's tool_calls were not parsed")
	assert.Equal(t, "call_x1", resp.ToolCalls[0].ID)
	assert.Equal(t, "get_weather", resp.ToolCalls[0].Name)
	assert.Equal(t, "Madison", resp.ToolCalls[0].Arguments["city"])
}

// The follow-up turn: the assistant's tool call and the tool result must reach
// Ollama, which names a result's tool with tool_name.
func TestOllama_ToolCallHistoryIsSent(t *testing.T) {
	f := &fakeOllama{capabilities: []string{"completion", "tools"}}
	p := ollamaFor(f.server(t))

	_, err := p.GenerateCompletion(context.Background(), &provider.CompletionRequest{
		Model: "qwen2.5:14b", Tools: []provider.Tool{weather},
		Messages: []provider.Message{
			{Role: "user", Content: "weather?"},
			{Role: "assistant", ToolCalls: []provider.ToolCall{{ID: "call_x1", Name: "get_weather", Arguments: map[string]interface{}{"city": "Madison"}}}},
			{Role: "tool", ToolCallID: "call_x1", Content: `{"temp_f":61}`},
		},
	})
	require.NoError(t, err)

	msgs := f.lastBody()["messages"].([]interface{})
	require.Len(t, msgs, 3)
	asst := msgs[1].(map[string]interface{})
	calls, ok := asst["tool_calls"].([]interface{})
	require.True(t, ok, "the assistant's tool_calls were not sent: %v", asst)
	fn := calls[0].(map[string]interface{})["function"].(map[string]interface{})
	assert.Equal(t, "get_weather", fn["name"])
	assert.Equal(t, "Madison", fn["arguments"].(map[string]interface{})["city"], "Ollama takes arguments as an object")
	tool := msgs[2].(map[string]interface{})
	assert.Equal(t, "tool", tool["role"])
	assert.Equal(t, "get_weather", tool["tool_name"], "the tool result must name its tool")
}

// Whether a model takes native tools is Ollama's answer (/api/show
// capabilities), not a hardcoded list. A model without "tools" goes through
// the text fallback, so no native tools field is sent.
func TestOllama_CapabilitiesDecideNativeVersusTextFallback(t *testing.T) {
	f := &fakeOllama{capabilities: []string{"completion"}}
	p := ollamaFor(f.server(t))
	_, err := p.GenerateCompletion(context.Background(), &provider.CompletionRequest{
		Model: "qwen2.5:14b", Messages: []provider.Message{{Role: "user", Content: "x"}}, Tools: []provider.Tool{weather},
	})
	require.NoError(t, err)
	_, hasTools := f.lastBody()["tools"]
	assert.False(t, hasTools, "a model whose capabilities lack tools must use the text fallback, even when a name list says otherwise")

	g := &fakeOllama{capabilities: []string{"completion", "tools"}}
	q := ollamaFor(g.server(t))
	for i := 0; i < 3; i++ {
		_, err = q.GenerateCompletion(context.Background(), &provider.CompletionRequest{
			Model: "some-new-model:7b", Messages: []provider.Message{{Role: "user", Content: "x"}}, Tools: []provider.Tool{weather},
		})
		require.NoError(t, err)
	}
	_, hasTools = g.lastBody()["tools"]
	assert.True(t, hasTools, "a model Ollama says takes tools gets them natively, even off any name list")
	assert.Equal(t, 1, g.shows, "capabilities are cached per model, not fetched per request")
}

// An Ollama too old to report capabilities keeps the old name-list behavior.
func TestOllama_NoCapabilitiesFieldFallsBackToTheNameList(t *testing.T) {
	f := &fakeOllama{capabilities: nil}
	p := ollamaFor(f.server(t))
	_, err := p.GenerateCompletion(context.Background(), &provider.CompletionRequest{
		Model: "llama3.2:3b", Messages: []provider.Message{{Role: "user", Content: "x"}}, Tools: []provider.Tool{weather},
	})
	require.NoError(t, err)
	_, hasTools := f.lastBody()["tools"]
	assert.True(t, hasTools, "llama3.2 is on the fallback list")
}
