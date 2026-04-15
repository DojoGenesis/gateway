package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/DojoGenesis/gateway/provider"
)

// GoogleProvider implements the Google Gemini API adapter.
// Uses query parameter authentication and a custom message format (contents/parts).
type GoogleProvider struct {
	BaseProvider
}

func NewGoogleProvider(apiKey string) *GoogleProvider {
	return &GoogleProvider{
		BaseProvider: BaseProvider{
			Name:       "google",
			BaseURL:    "https://generativelanguage.googleapis.com/v1beta",
			APIKey:     apiKey,
			Client:     NewHTTPClient(),
			EnvKeyName: "GOOGLE_API_KEY",
		},
	}
}

func (p *GoogleProvider) GetInfo(ctx context.Context) (*provider.ProviderInfo, error) {
	return &provider.ProviderInfo{
		Name:         "google",
		Version:      "1.0.0",
		Description:  "Google Gemini API provider (in-process)",
		Capabilities: []string{"completion", "streaming", "tool_calling"},
	}, nil
}

func (p *GoogleProvider) ListModels(ctx context.Context) ([]provider.ModelInfo, error) {
	return []provider.ModelInfo{
		{ID: "gemini-2.0-flash", Name: "Gemini 2.0 Flash", Provider: "google", ContextSize: 1048576, Cost: 0.10},
		{ID: "gemini-2.0-pro", Name: "Gemini 2.0 Pro", Provider: "google", ContextSize: 2097152, Cost: 1.25},
		{ID: "gemini-1.5-flash", Name: "Gemini 1.5 Flash", Provider: "google", ContextSize: 1048576, Cost: 0.075},
	}, nil
}

// --- Gemini API types ---

type geminiRequest struct {
	Contents         []geminiContent        `json:"contents"`
	GenerationConfig *geminiGenerationCfg   `json:"generationConfig,omitempty"`
	Tools            []geminiToolDecl       `json:"tools,omitempty"`
	SystemInstruction *geminiContent        `json:"systemInstruction,omitempty"`
}

type geminiContent struct {
	Role  string       `json:"role,omitempty"`
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text         string              `json:"text,omitempty"`
	FunctionCall *geminiFunctionCall `json:"functionCall,omitempty"`
	FunctionResp *geminiFunctionResp `json:"functionResponse,omitempty"`
}

type geminiFunctionCall struct {
	Name string                 `json:"name"`
	Args map[string]interface{} `json:"args"`
}

type geminiFunctionResp struct {
	Name     string      `json:"name"`
	Response interface{} `json:"response"`
}

type geminiGenerationCfg struct {
	Temperature     float64 `json:"temperature,omitempty"`
	MaxOutputTokens int     `json:"maxOutputTokens,omitempty"`
}

type geminiToolDecl struct {
	FunctionDeclarations []geminiFunctionDecl `json:"functionDeclarations"`
}

type geminiFunctionDecl struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Parameters  interface{} `json:"parameters,omitempty"`
}

type geminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []geminiPart `json:"parts"`
			Role  string      `json:"role"`
		} `json:"content"`
		FinishReason string `json:"finishReason"`
	} `json:"candidates"`
	UsageMetadata struct {
		PromptTokenCount     int `json:"promptTokenCount"`
		CandidatesTokenCount int `json:"candidatesTokenCount"`
		TotalTokenCount      int `json:"totalTokenCount"`
	} `json:"usageMetadata"`
}

// doGeminiRequest handles the Gemini-specific auth (API key as query param).
func (p *GoogleProvider) doGeminiRequest(ctx context.Context, method, url string, body io.Reader) (*http.Response, error) {
	apiKey := p.ResolveAPIKey(ctx)
	if apiKey == "" {
		return nil, fmt.Errorf("GOOGLE_API_KEY is not set")
	}

	// Append API key as query parameter
	separator := "?"
	if len(url) > 0 {
		for _, c := range url {
			if c == '?' {
				separator = "&"
				break
			}
		}
	}
	fullURL := url + separator + "key=" + apiKey

	req, err := http.NewRequestWithContext(ctx, method, fullURL, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("API request to google failed: %w", err)
	}

	if resp.StatusCode >= 400 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("google API returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	return resp, nil
}

func (p *GoogleProvider) GenerateCompletion(ctx context.Context, req *provider.CompletionRequest) (*provider.CompletionResponse, error) {
	model := req.Model
	if model == "" {
		model = "gemini-2.0-flash"
	}

	gReq := p.buildGeminiRequest(req)

	body, _ := json.Marshal(gReq)
	url := fmt.Sprintf("%s/models/%s:generateContent", p.BaseURL, model)

	resp, err := p.doGeminiRequest(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var gResp geminiResponse
	if err := json.NewDecoder(resp.Body).Decode(&gResp); err != nil {
		return nil, fmt.Errorf("failed to decode Gemini response: %w", err)
	}

	return p.parseGeminiResponse(&gResp, model), nil
}

func (p *GoogleProvider) GenerateCompletionStream(ctx context.Context, req *provider.CompletionRequest) (<-chan *provider.CompletionChunk, error) {
	model := req.Model
	if model == "" {
		model = "gemini-2.0-flash"
	}

	gReq := p.buildGeminiRequest(req)

	body, _ := json.Marshal(gReq)
	url := fmt.Sprintf("%s/models/%s:streamGenerateContent?alt=sse", p.BaseURL, model)

	resp, err := p.doGeminiRequest(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	ch := make(chan *provider.CompletionChunk, 100)
	go func() {
		defer close(ch)
		dataCh := make(chan string, 100)
		go p.StreamSSE(ctx, resp, dataCh)

		for data := range dataCh {
			var gResp geminiResponse
			if err := json.Unmarshal([]byte(data), &gResp); err != nil {
				continue
			}
			if len(gResp.Candidates) > 0 && len(gResp.Candidates[0].Content.Parts) > 0 {
				for _, part := range gResp.Candidates[0].Content.Parts {
					if part.Text != "" {
						ch <- &provider.CompletionChunk{Delta: part.Text, Done: false}
					}
				}
				if gResp.Candidates[0].FinishReason == "STOP" {
					ch <- &provider.CompletionChunk{Done: true}
					return
				}
			}
		}
		ch <- &provider.CompletionChunk{Done: true}
	}()
	return ch, nil
}

func (p *GoogleProvider) CallTool(ctx context.Context, req *provider.ToolCallRequest) (*provider.ToolCallResponse, error) {
	return &provider.ToolCallResponse{
		Error: fmt.Sprintf("tool execution not supported by Google provider; tool '%s' should be executed by the agent layer", req.ToolCall.Name),
	}, nil
}

func (p *GoogleProvider) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	reqBody := map[string]interface{}{
		"content": map[string]interface{}{
			"parts": []map[string]interface{}{
				{"text": text},
			},
		},
	}
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("google: marshal embedding request: %w", err)
	}

	url := fmt.Sprintf("%s/models/gemini-embedding-001:embedContent", p.BaseURL)
	resp, err := p.doGeminiRequest(ctx, "POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("google: embedding request failed: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Embedding struct {
			Values []float32 `json:"values"`
		} `json:"embedding"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("google: decode embedding response: %w", err)
	}
	if len(result.Embedding.Values) == 0 {
		return nil, fmt.Errorf("google: empty embedding response")
	}
	return result.Embedding.Values, nil
}

// buildGeminiRequest converts a Gateway CompletionRequest to a Gemini API request.
func (p *GoogleProvider) buildGeminiRequest(req *provider.CompletionRequest) geminiRequest {
	gReq := geminiRequest{}

	// Extract system instruction and convert messages
	var contents []geminiContent
	for _, m := range req.Messages {
		if m.Role == "system" {
			gReq.SystemInstruction = &geminiContent{
				Parts: []geminiPart{{Text: m.Content}},
			}
			continue
		}

		role := m.Role
		if role == "assistant" {
			role = "model"
		}

		contents = append(contents, geminiContent{
			Role:  role,
			Parts: []geminiPart{{Text: m.Content}},
		})
	}
	gReq.Contents = contents

	if req.Temperature > 0 || req.MaxTokens > 0 {
		gReq.GenerationConfig = &geminiGenerationCfg{
			Temperature:     req.Temperature,
			MaxOutputTokens: req.MaxTokens,
		}
	}

	if len(req.Tools) > 0 {
		decls := make([]geminiFunctionDecl, len(req.Tools))
		for i, t := range req.Tools {
			decls[i] = geminiFunctionDecl{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  t.Parameters,
			}
		}
		gReq.Tools = []geminiToolDecl{{FunctionDeclarations: decls}}
	}

	return gReq
}

// parseGeminiResponse converts a Gemini API response to a Gateway CompletionResponse.
func (p *GoogleProvider) parseGeminiResponse(gResp *geminiResponse, model string) *provider.CompletionResponse {
	var content string
	var toolCalls []provider.ToolCall

	if len(gResp.Candidates) > 0 {
		for _, part := range gResp.Candidates[0].Content.Parts {
			if part.Text != "" {
				content += part.Text
			}
			if part.FunctionCall != nil {
				toolCalls = append(toolCalls, provider.ToolCall{
					Name:      part.FunctionCall.Name,
					Arguments: part.FunctionCall.Args,
				})
			}
		}
	}

	return &provider.CompletionResponse{
		Model:   model,
		Content: content,
		Usage: provider.Usage{
			InputTokens:  gResp.UsageMetadata.PromptTokenCount,
			OutputTokens: gResp.UsageMetadata.CandidatesTokenCount,
			TotalTokens:  gResp.UsageMetadata.TotalTokenCount,
		},
		ToolCalls: toolCalls,
	}
}
