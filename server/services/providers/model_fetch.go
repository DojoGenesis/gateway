package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/DojoGenesis/gateway/provider"
)

// modelCacheTTL bounds how long a dynamically-fetched /models list is reused
// before the provider re-queries upstream. Fetching on every ListModels call
// would add a network round-trip to /v1/models and to internal model listings.
const modelCacheTTL = time.Hour

// modelCache is a concurrency-safe TTL cache of dynamically-fetched models.
// Embed it (by value) in a provider struct; the zero value is ready to use.
type modelCache struct {
	mu      sync.Mutex
	models  []provider.ModelInfo
	fetched time.Time
}

func (c *modelCache) get() ([]provider.ModelInfo, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.models) > 0 && time.Since(c.fetched) < modelCacheTTL {
		return c.models, true
	}
	return nil, false
}

func (c *modelCache) set(models []provider.ModelInfo) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.models = models
	c.fetched = time.Now()
}

// listModelsDynamic is the shared "fetch live, fall back to static" wrapper used
// by every provider whose upstream exposes a model-list endpoint.
//
//   - Fresh cache  -> return it.
//   - fetch() ok   -> enrich with curated static metadata, cache, return it.
//     The LIVE list is authoritative for WHICH models exist (this is the whole
//     point — it never goes stale); the static list only supplies metadata
//     (pricing, friendly names) that /models endpoints often omit.
//   - fetch() fails or returns nothing -> return the static list unchanged, so a
//     configured provider is never left without a catalog when its endpoint is
//     briefly unreachable.
func listModelsDynamic(
	ctx context.Context,
	cache *modelCache,
	static []provider.ModelInfo,
	fetch func(context.Context) ([]provider.ModelInfo, error),
) []provider.ModelInfo {
	if cached, ok := cache.get(); ok {
		return cached
	}
	fetched, err := fetch(ctx)
	if err != nil {
		return static
	}
	fetched = filterChatModels(fetched)
	if len(fetched) == 0 {
		return static
	}
	fetched = mergeStaticMetadata(fetched, static)
	cache.set(fetched)
	return fetched
}

// filterChatModels drops non-chat models (embeddings, TTS, transcription,
// image, moderation, legacy base-completion, realtime) from a fetched list, so
// a chat gateway never advertises a model that cannot serve /chat/completions.
// Applied uniformly to every provider's live list. See isNonChatModel.
func filterChatModels(models []provider.ModelInfo) []provider.ModelInfo {
	out := models[:0]
	for _, m := range models {
		if isNonChatModel(m.ID) {
			continue
		}
		out = append(out, m)
	}
	return out
}

// mergeStaticMetadata copies curated Name/ContextSize/Cost from the static list
// onto dynamically-fetched models whose IDs match. The dynamic entries stay
// authoritative for existence; only empty/zero metadata fields are filled in.
func mergeStaticMetadata(dynamic, static []provider.ModelInfo) []provider.ModelInfo {
	byID := make(map[string]provider.ModelInfo, len(static))
	for _, m := range static {
		byID[m.ID] = m
	}
	for i, d := range dynamic {
		s, ok := byID[d.ID]
		if !ok {
			continue
		}
		if d.Name == "" || d.Name == d.ID {
			dynamic[i].Name = s.Name
		}
		if d.ContextSize == 0 {
			dynamic[i].ContextSize = s.ContextSize
		}
		if d.Cost == 0 {
			dynamic[i].Cost = s.Cost
		}
	}
	return dynamic
}

// --- OpenAI-format /models fetcher ---------------------------------------------
// Used by every OpenAI-compatible provider: openai, groq, mistral, deepseek,
// kimi, and openrouter. OpenRouter additionally returns `name` and
// `context_length`, which we pick up when present.

type openAIModelsResponse struct {
	Data []struct {
		ID            string `json:"id"`
		Name          string `json:"name"`
		ContextLength int    `json:"context_length"`
	} `json:"data"`
}

func fetchOpenAIFormatModels(ctx context.Context, b *BaseProvider) ([]provider.ModelInfo, error) {
	apiKey := b.ResolveAPIKey(ctx)
	if apiKey == "" {
		return nil, fmt.Errorf("%s: no API key for model list", b.Name)
	}
	resp, err := b.DoRequest(ctx, "GET", "/models", nil, map[string]string{
		"Authorization": "Bearer " + apiKey,
	})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var out openAIModelsResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("%s: decode /models: %w", b.Name, err)
	}
	models := make([]provider.ModelInfo, 0, len(out.Data))
	for _, m := range out.Data {
		if m.ID == "" {
			continue
		}
		name := m.Name
		if name == "" {
			name = m.ID
		}
		models = append(models, provider.ModelInfo{
			ID: m.ID, Name: name, Provider: b.Name, ContextSize: m.ContextLength,
		})
	}
	return models, nil
}

// isNonChatModel reports whether an OpenAI-format model ID names a non-chat
// model — embeddings, audio (TTS/transcription), images, moderation, the legacy
// base-completion models, or the realtime API — none of which serve
// /chat/completions. A chat gateway's model list should not advertise them.
// Applies across the OpenAI-compatible family (OpenAI, Groq's whisper, etc.).
func isNonChatModel(id string) bool {
	lid := strings.ToLower(id)
	for _, marker := range []string{
		"embedding", "tts", "whisper", "transcribe", "dall-e",
		"moderation", "davinci", "babbage", "gpt-image", "-realtime",
	} {
		if strings.Contains(lid, marker) {
			return true
		}
	}
	return false
}

// --- Anthropic /v1/models fetcher ----------------------------------------------

func fetchAnthropicModels(ctx context.Context, b *BaseProvider) ([]provider.ModelInfo, error) {
	apiKey := b.ResolveAPIKey(ctx)
	if apiKey == "" {
		return nil, fmt.Errorf("anthropic: no API key for model list")
	}
	resp, err := b.DoRequest(ctx, "GET", "/v1/models?limit=1000", nil, map[string]string{
		"x-api-key":         apiKey,
		"anthropic-version": "2023-06-01",
	})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var out struct {
		Data []struct {
			ID          string `json:"id"`
			DisplayName string `json:"display_name"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("anthropic: decode /v1/models: %w", err)
	}
	models := make([]provider.ModelInfo, 0, len(out.Data))
	for _, m := range out.Data {
		if m.ID == "" {
			continue
		}
		name := m.DisplayName
		if name == "" {
			name = m.ID
		}
		// Anthropic's list endpoint does not report a context window; the
		// current generation is 200k, and mergeStaticMetadata refines known IDs.
		models = append(models, provider.ModelInfo{
			ID: m.ID, Name: name, Provider: "anthropic", ContextSize: 200000,
		})
	}
	return models, nil
}

// --- Google (Gemini) /models fetcher -------------------------------------------

func fetchGoogleModels(ctx context.Context, b *BaseProvider) ([]provider.ModelInfo, error) {
	apiKey := b.ResolveAPIKey(ctx)
	if apiKey == "" {
		return nil, fmt.Errorf("google: no API key for model list")
	}
	// Gemini authenticates the model list via ?key=; pageSize maxes at 1000.
	path := "/models?pageSize=1000&key=" + url.QueryEscape(apiKey)
	resp, err := b.DoRequest(ctx, "GET", path, nil, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var out struct {
		Models []struct {
			Name                       string   `json:"name"` // "models/gemini-2.5-pro"
			DisplayName                string   `json:"displayName"`
			InputTokenLimit            int      `json:"inputTokenLimit"`
			SupportedGenerationMethods []string `json:"supportedGenerationMethods"`
		} `json:"models"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("google: decode /models: %w", err)
	}
	models := make([]provider.ModelInfo, 0, len(out.Models))
	for _, m := range out.Models {
		// Only surface models that can actually serve chat completions.
		if !containsString(m.SupportedGenerationMethods, "generateContent") {
			continue
		}
		id := strings.TrimPrefix(m.Name, "models/")
		if id == "" {
			continue
		}
		name := m.DisplayName
		if name == "" {
			name = id
		}
		models = append(models, provider.ModelInfo{
			ID: id, Name: name, Provider: "google", ContextSize: m.InputTokenLimit,
		})
	}
	return models, nil
}

func containsString(ss []string, target string) bool {
	for _, s := range ss {
		if s == target {
			return true
		}
	}
	return false
}
