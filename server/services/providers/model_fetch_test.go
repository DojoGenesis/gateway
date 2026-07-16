package providers

import (
	"context"
	"errors"
	"testing"

	"github.com/DojoGenesis/gateway/provider"
)

func TestIsNonChatModel(t *testing.T) {
	nonChat := []string{
		"text-embedding-3-small", "text-embedding-ada-002", "whisper-1",
		"tts-1", "gpt-4o-mini-tts", "gpt-4o-transcribe", "dall-e-3",
		"omni-moderation-latest", "davinci-002", "babbage-002",
		"gpt-image-1", "gpt-4o-realtime-preview",
	}
	chat := []string{
		"gpt-4o", "gpt-4.1", "o3", "o4-mini", "o1-pro", "gpt-4o-mini",
		"gpt-4o-mini-search-preview", // search IS chat-capable
		"llama-3.3-70b-versatile", "deepseek-chat", "kimi-k2.5",
		"openai/gpt-4o", "anthropic/claude-sonnet-4.5",
	}
	for _, id := range nonChat {
		if !isNonChatModel(id) {
			t.Errorf("isNonChatModel(%q) = false, want true (should be filtered out)", id)
		}
	}
	for _, id := range chat {
		if isNonChatModel(id) {
			t.Errorf("isNonChatModel(%q) = true, want false (a chat model was filtered out)", id)
		}
	}
}

func TestListModelsDynamic_FallsBackToStaticOnError(t *testing.T) {
	static := []provider.ModelInfo{{ID: "static-1", Provider: "x"}}
	var cache modelCache
	got := listModelsDynamic(context.Background(), &cache, static,
		func(context.Context) ([]provider.ModelInfo, error) {
			return nil, errors.New("upstream down")
		})
	if len(got) != 1 || got[0].ID != "static-1" {
		t.Fatalf("expected static fallback on fetch error, got %+v", got)
	}
	// A failed fetch must NOT poison the cache.
	if _, ok := cache.get(); ok {
		t.Fatalf("cache should be empty after a failed fetch")
	}
}

func TestListModelsDynamic_FallsBackOnEmpty(t *testing.T) {
	static := []provider.ModelInfo{{ID: "static-1", Provider: "x"}}
	var cache modelCache
	got := listModelsDynamic(context.Background(), &cache, static,
		func(context.Context) ([]provider.ModelInfo, error) {
			return []provider.ModelInfo{}, nil // success but empty
		})
	if len(got) != 1 || got[0].ID != "static-1" {
		t.Fatalf("expected static fallback on empty fetch, got %+v", got)
	}
}

func TestListModelsDynamic_LiveIsAuthoritativeAndCached(t *testing.T) {
	static := []provider.ModelInfo{
		{ID: "gpt-stale", Provider: "openai", Cost: 1.0},         // retired: must NOT appear
		{ID: "gpt-live", Name: "GPT Live", Provider: "openai", Cost: 5.0, ContextSize: 128000}, // metadata source
	}
	calls := 0
	fetch := func(context.Context) ([]provider.ModelInfo, error) {
		calls++
		return []provider.ModelInfo{{ID: "gpt-live", Provider: "openai"}}, nil // live: id only
	}
	var cache modelCache
	got := listModelsDynamic(context.Background(), &cache, static, fetch)

	if len(got) != 1 || got[0].ID != "gpt-live" {
		t.Fatalf("live list must be authoritative (no stale IDs), got %+v", got)
	}
	// Metadata merged from the static entry with the matching ID.
	if got[0].Name != "GPT Live" || got[0].Cost != 5.0 || got[0].ContextSize != 128000 {
		t.Fatalf("expected static metadata merged onto live model, got %+v", got[0])
	}
	// Second call must be served from cache (no extra fetch).
	_ = listModelsDynamic(context.Background(), &cache, static, fetch)
	if calls != 1 {
		t.Fatalf("expected fetch to run once (cached thereafter), ran %d times", calls)
	}
}
