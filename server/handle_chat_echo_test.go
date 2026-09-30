package server

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/DojoGenesis/gateway/provider"
)

// DGS-141, second half: a caller that logs every model input could not see
// text the gateway prepended. Every response now says what was injected
// (header, always), and a caller that asks gets the exact text back.

func echoProvider() *capturingProvider {
	return &capturingProvider{name: "p1", models: []provider.ModelInfo{{ID: "m1"}}}
}

func firstSSEData(t *testing.T, body string) map[string]interface{} {
	t.Helper()
	for _, line := range strings.Split(body, "\n") {
		if strings.HasPrefix(line, "data: {") {
			var chunk map[string]interface{}
			require.NoError(t, json.Unmarshal([]byte(strings.TrimPrefix(line, "data: ")), &chunk))
			return chunk
		}
	}
	t.Fatalf("no SSE data chunk in:\n%s", body)
	return nil
}

func TestEcho_HeaderNamesWhatWasInjected(t *testing.T) {
	t.Setenv("SYSTEM_PROMPT", "BASE PROMPT")
	t.Setenv("SYSTEM_PROMPT_FILE", "")
	for _, stream := range []string{"false", "true"} {
		for _, tc := range []struct {
			name, messages string
			headers        map[string]string
			want           string
		}{
			{"injected", `[{"role":"user","content":"hi"}]`, nil, "base_system_prompt"},
			{"caller system message", `[{"role":"system","content":"mine"},{"role":"user","content":"hi"}]`, nil, "none"},
			{"X-Route direct", `[{"role":"user","content":"hi"}]`, map[string]string{"X-Route": "direct"}, "none"},
		} {
			w := postChat(chatServer(echoProvider()), `{"model":"m1","stream":`+stream+`,"messages":`+tc.messages+`}`, tc.headers)
			require.Equal(t, http.StatusOK, w.Code, w.Body.String())
			assert.Equal(t, tc.want, w.Header().Get("X-Dojo-Injected"), "stream=%s %s", stream, tc.name)
		}
	}
}

func TestEcho_NonStreamReturnsTheExactInjectedTextOnRequest(t *testing.T) {
	t.Setenv("SYSTEM_PROMPT", "BASE PROMPT")
	t.Setenv("SYSTEM_PROMPT_FILE", "")
	p := echoProvider()
	w := postChat(chatServer(p), `{"model":"m1","messages":[{"role":"user","content":"hi"}]}`, map[string]string{"X-Dojo-Echo-Injected": "1"})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	inj, ok := resp["dojo_injected"].([]interface{})
	require.True(t, ok, "no dojo_injected in the response: %s", w.Body.String())
	require.Len(t, inj, 1)
	first := inj[0].(map[string]interface{})
	assert.Equal(t, "base_system_prompt", first["source"])
	assert.Equal(t, "system", first["role"])
	assert.Equal(t, "BASE PROMPT", first["content"])
	// And it is exactly what the model received.
	assert.Equal(t, p.last().Messages[0].Content, first["content"])
}

func TestEcho_StreamPutsTheInjectedTextInTheFirstChunk(t *testing.T) {
	t.Setenv("SYSTEM_PROMPT", "BASE PROMPT")
	t.Setenv("SYSTEM_PROMPT_FILE", "")
	w := postChat(chatServer(echoProvider()), `{"model":"m1","stream":true,"messages":[{"role":"user","content":"hi"}]}`, map[string]string{"X-Dojo-Echo-Injected": "1"})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	inj, ok := firstSSEData(t, w.Body.String())["dojo_injected"].([]interface{})
	require.True(t, ok, "first chunk has no dojo_injected:\n%s", w.Body.String())
	assert.Equal(t, "BASE PROMPT", inj[0].(map[string]interface{})["content"])
}

func TestEcho_ToolStreamAlsoEchoes(t *testing.T) {
	t.Setenv("SYSTEM_PROMPT", "BASE PROMPT")
	t.Setenv("SYSTEM_PROMPT_FILE", "")
	p := &toolProvider{calls: []provider.ToolCall{{ID: "c", Name: "get_weather", Arguments: map[string]interface{}{}}}}
	w := postChat(toolServer(p), `{"model":"m1","stream":true,"messages":[{"role":"user","content":"x"}],"tools":[`+weatherTool+`]}`, map[string]string{"X-Dojo-Echo-Injected": "1"})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, "base_system_prompt", w.Header().Get("X-Dojo-Injected"))
	_, ok := firstSSEData(t, w.Body.String())["dojo_injected"]
	assert.True(t, ok, "the tool-stream path must echo too")
}

// Without the opt-in, the body is unchanged: no extra field for clients that
// never asked. With nothing injected, an opt-in gets an empty list, not
// silence, so "nothing was added" is itself an answer.
func TestEcho_BodyFieldOnlyOnRequest(t *testing.T) {
	t.Setenv("SYSTEM_PROMPT", "BASE PROMPT")
	t.Setenv("SYSTEM_PROMPT_FILE", "")
	w := postChat(chatServer(echoProvider()), `{"model":"m1","messages":[{"role":"user","content":"hi"}]}`, nil)
	assert.NotContains(t, w.Body.String(), "dojo_injected")

	w = postChat(chatServer(echoProvider()), `{"model":"m1","messages":[{"role":"user","content":"hi"}]}`, map[string]string{"X-Route": "direct", "X-Dojo-Echo-Injected": "1"})
	assert.Contains(t, w.Body.String(), `"dojo_injected":[]`)
}
