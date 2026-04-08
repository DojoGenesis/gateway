//go:build integration

package providers

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/DojoGenesis/gateway/provider"
)

func skipIfNoKey(t *testing.T, envKey string) {
	if os.Getenv("INTEGRATION_TEST") != "true" {
		t.Skip("INTEGRATION_TEST not set")
	}
	if os.Getenv(envKey) == "" {
		t.Skipf("%s not set, skipping", envKey)
	}
}

func integrationCompletionTest(t *testing.T, p provider.ModelProvider, name string) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Non-streaming completion
	resp, err := p.GenerateCompletion(ctx, &provider.CompletionRequest{
		Messages:  []provider.Message{{Role: "user", Content: "Hello, respond in one word."}},
		MaxTokens: 10,
	})
	if err != nil {
		t.Fatalf("%s completion failed: %v", name, err)
	}
	if resp.Content == "" {
		t.Errorf("%s returned empty content", name)
	}
	t.Logf("%s completion: %q", name, resp.Content)

	// Streaming completion
	ch, err := p.GenerateCompletionStream(ctx, &provider.CompletionRequest{
		Messages:  []provider.Message{{Role: "user", Content: "Hello, respond in one word."}},
		MaxTokens: 10,
	})
	if err != nil {
		t.Fatalf("%s streaming failed: %v", name, err)
	}

	var content string
	for chunk := range ch {
		content += chunk.Delta
		if chunk.Done {
			break
		}
	}
	if content == "" {
		t.Errorf("%s streaming returned no content", name)
	}
	t.Logf("%s streaming: %q", name, content)
}

func TestIntegration_Anthropic(t *testing.T) {
	skipIfNoKey(t, "ANTHROPIC_API_KEY")
	p := NewAnthropicProvider("")
	integrationCompletionTest(t, p, "anthropic")
}

func TestIntegration_OpenAI(t *testing.T) {
	skipIfNoKey(t, "OPENAI_API_KEY")
	p := NewOpenAIProvider("")
	integrationCompletionTest(t, p, "openai")
}

func TestIntegration_Google(t *testing.T) {
	skipIfNoKey(t, "GOOGLE_API_KEY")
	p := NewGoogleProvider("")
	integrationCompletionTest(t, p, "google")
}

func TestIntegration_Groq(t *testing.T) {
	skipIfNoKey(t, "GROQ_API_KEY")
	p := NewGroqProvider("")
	integrationCompletionTest(t, p, "groq")
}

func TestIntegration_Mistral(t *testing.T) {
	skipIfNoKey(t, "MISTRAL_API_KEY")
	p := NewMistralProvider("")
	integrationCompletionTest(t, p, "mistral")
}

func TestIntegration_DeepSeek(t *testing.T) {
	skipIfNoKey(t, "DEEPSEEK_API_KEY")
	p := NewDeepSeekProvider("")
	integrationCompletionTest(t, p, "deepseek-api")
}

func TestIntegration_Kimi(t *testing.T) {
	skipIfNoKey(t, "KIMI_API_KEY")
	p := NewKimiProvider("")
	integrationCompletionTest(t, p, "kimi")
}

func TestIntegration_Ollama(t *testing.T) {
	if os.Getenv("INTEGRATION_TEST") != "true" {
		t.Skip("INTEGRATION_TEST not set")
	}
	p := NewOllamaProvider()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if !p.IsAvailable(ctx) {
		t.Skip("Ollama not available")
	}
	integrationCompletionTest(t, p, "ollama")
}
