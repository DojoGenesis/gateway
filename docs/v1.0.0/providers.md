# Provider Layer v1.0.0

The provider layer manages connections to LLM APIs. Providers are auto-discovered at
startup: cloud providers register if their API key is available, local providers register
if their service is reachable.

## Supported Providers

| Provider | Type | Auth | Env Key | Streaming |
|----------|------|------|---------|-----------|
| Anthropic | Cloud | `x-api-key` header | `ANTHROPIC_API_KEY` | SSE (`content_block_delta`) |
| OpenAI | Cloud | Bearer token | `OPENAI_API_KEY` | SSE (OpenAI format) |
| Google (Gemini) | Cloud | Query param `?key=` | `GOOGLE_API_KEY` | SSE (Gemini format) |
| Groq | Cloud | Bearer token | `GROQ_API_KEY` | SSE (OpenAI format) |
| Mistral | Cloud | Bearer token | `MISTRAL_API_KEY` | SSE (OpenAI format) |
| DeepSeek | Cloud | Bearer token | `DEEPSEEK_API_KEY` | SSE (OpenAI format) |
| Kimi (K2.5) | Cloud | Bearer token | `KIMI_API_KEY` | SSE (OpenAI format) |
| Ollama | Local | None | `OLLAMA_HOST` | NDJSON |

## Quick Start

1. Set one or more API keys in your environment:
   ```bash
   export ANTHROPIC_API_KEY=sk-ant-...
   export OPENAI_API_KEY=sk-...
   ```

2. Start the gateway — providers with valid keys auto-register:
   ```bash
   go run .
   ```

3. Check what loaded:
   ```bash
   curl http://localhost:8081/admin/providers | jq .
   ```

## Architecture

```
BaseProvider (base.go)
├── openaiCompatibleProvider (openai_compat.go)
│   ├── OpenAIProvider
│   ├── GroqProvider
│   ├── MistralProvider
│   ├── DeepSeekProvider
│   └── KimiProvider
├── AnthropicProvider (anthropic.go)      — Messages API
├── GoogleProvider (google.go)            — Gemini API
└── OllamaProvider (ollama.go)            — /api/chat NDJSON
```

`BaseProvider` provides: HTTP client, API key resolution, SSE streaming, request helpers.

`openaiCompatibleProvider` adds: shared OpenAI Chat Completions format (request/response
types, message conversion, tool calling) so providers that follow this format need only
configure their URL, models, and API key.

## API Key Resolution

Keys are resolved in this order (first non-empty wins):

1. **KeyResolver function** — dynamic per-request resolution (e.g., from user session)
2. **Static APIKey** — set at provider construction
3. **Environment variable** — e.g., `ANTHROPIC_API_KEY`

## Environment Variables

### Cloud Provider Keys
| Variable | Provider |
|----------|----------|
| `ANTHROPIC_API_KEY` | Anthropic |
| `OPENAI_API_KEY` | OpenAI |
| `GOOGLE_API_KEY` | Google Gemini |
| `GROQ_API_KEY` | Groq |
| `MISTRAL_API_KEY` | Mistral |
| `DEEPSEEK_API_KEY` | DeepSeek |
| `KIMI_API_KEY` | Kimi |

### Base URL Overrides
| Variable | Default |
|----------|---------|
| `OPENAI_BASE_URL` | `https://api.openai.com/v1` |
| `DEEPSEEK_BASE_URL` | `https://api.deepseek.com` |
| `KIMI_BASE_URL` | `https://api.moonshot.ai/v1` |

### Local Provider Config
| Variable | Default |
|----------|---------|
| `OLLAMA_HOST` | `http://localhost:11434` |
| `OLLAMA_DEFAULT_MODEL` | `llama3.2` |

## Admin Endpoint

`GET /admin/providers` returns the live status of all registered providers:

```json
{
  "providers": [
    {
      "name": "anthropic",
      "version": "1.0.0",
      "description": "Anthropic Messages API provider (in-process)",
      "capabilities": ["completion", "streaming", "tool_calling"],
      "models": [
        {"id": "claude-sonnet-4-20250514", "name": "Claude Sonnet 4", "context_size": 200000}
      ]
    }
  ],
  "total": 3
}
```

## Adding a New Provider

For OpenAI-compatible APIs, create a new file in `server/services/providers/`:

```go
type MyProvider struct {
    openaiCompatibleProvider
}

func NewMyProvider(apiKey string) *MyProvider {
    return &MyProvider{
        openaiCompatibleProvider: openaiCompatibleProvider{
            BaseProvider: BaseProvider{
                Name:       "my-provider",
                BaseURL:    "https://api.my-provider.com/v1",
                APIKey:     apiKey,
                Client:     NewHTTPClient(),
                EnvKeyName: "MY_PROVIDER_API_KEY",
            },
            defaultModel: "my-model",
            models: []provider.ModelInfo{
                {ID: "my-model", Name: "My Model", Provider: "my-provider", ContextSize: 128000},
            },
            info: &provider.ProviderInfo{
                Name:         "my-provider",
                Version:      "1.0.0",
                Description:  "My provider (in-process)",
                Capabilities: []string{"completion", "streaming"},
            },
        },
    }
}
```

Then add it to `provider_registry.go` in the `cloudProviders` slice.

## Testing

```bash
# Unit + conformance tests (no API keys needed)
go test ./server/services/providers/... -v

# Integration tests (requires real API keys)
INTEGRATION_TEST=true go test ./server/services/providers/... -tags integration -v
```
