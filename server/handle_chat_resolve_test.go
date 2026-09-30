package server

import "testing"

import "github.com/DojoGenesis/gateway/provider"

// A deepseek-* model that no provider lists must still route to the DeepSeek
// provider by prefix. The prefix table said "deepseek", but RegisterProviders
// registers it as "deepseek-api" (services/provider_registry.go), so the
// prefix could never match and the request fell through to "first available",
// which Go map iteration makes a random pick.
func TestResolveProvider_DeepSeekPrefixReachesTheRegisteredProvider(t *testing.T) {
	ds := &capturingProvider{name: "deepseek-api"}
	others := []*capturingProvider{
		{name: "anthropic"}, {name: "openai"}, {name: "groq"}, {name: "kimi"}, {name: "mistral"},
	}
	s := chatServer(append(others, ds)...)
	for i := 0; i < 20; i++ { // map iteration order varies run to run
		name, _, err := s.resolveProvider("deepseek-chat")
		if err != nil {
			t.Fatal(err)
		}
		if name != "deepseek-api" {
			t.Fatalf("deepseek-chat resolved to %q, want deepseek-api", name)
		}
	}
}

// When nothing matches, the fallback must be the same provider every time.
func TestResolveProvider_FallbackIsDeterministic(t *testing.T) {
	s := chatServer(&capturingProvider{name: "zeta"}, &capturingProvider{name: "alpha"}, &capturingProvider{name: "mid"})
	for i := 0; i < 20; i++ {
		name, _, err := s.resolveProvider("no-such-model")
		if err != nil {
			t.Fatal(err)
		}
		if name != "alpha" {
			t.Fatalf("fallback picked %q, want the first by name (alpha)", name)
		}
	}
}

// An exact ListModels match wins over any prefix rule.
func TestResolveProvider_ExactModelMatchWins(t *testing.T) {
	or := &capturingProvider{name: "openrouter", models: []provider.ModelInfo{{ID: "deepseek/deepseek-v4-pro"}}}
	ds := &capturingProvider{name: "deepseek-api"}
	s := chatServer(or, ds)
	name, _, err := s.resolveProvider("deepseek/deepseek-v4-pro")
	if err != nil || name != "openrouter" {
		t.Fatalf("resolved to %q (err %v), want openrouter", name, err)
	}
}
