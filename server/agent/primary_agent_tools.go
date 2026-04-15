package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/DojoGenesis/gateway/memory"
	providerpkg "github.com/DojoGenesis/gateway/provider"
	"github.com/DojoGenesis/gateway/server/trace"
	"github.com/DojoGenesis/gateway/tools"
)

// ToolExecutionResult represents the result of executing a single tool call.
type ToolExecutionResult struct {
	ToolCallID string
	ToolName   string
	Result     map[string]interface{}
	Error      error
}

// QueryRequest represents a structured query request with all parameters.
type QueryRequest struct {
	Query         string
	UserID        string
	UserTier      string
	ProviderName  string
	ModelID       string
	Temperature   float64
	MaxTokens     int
	UseMemory     bool
	ProjectID     string // v0.0.18: Project context for scoped tools and memory
	WorkspaceRoot string // Absolute path to the user's workspace; file tools resolve relative paths against this
	// SystemPromptOverride replaces buildSystemPrompt output when non-empty.
	SystemPromptOverride string
}

// convertToolsToPluginFormat converts tool definitions to plugin Tool format.
func convertToolsToPluginFormat(toolDefs []*tools.ToolDefinition) []providerpkg.Tool {
	pluginTools := make([]providerpkg.Tool, len(toolDefs))
	for i, td := range toolDefs {
		pluginTools[i] = providerpkg.Tool{
			Name:        td.Name,
			Description: td.Description,
			Parameters:  td.Parameters,
		}
	}
	return pluginTools
}

// executeToolCalls executes multiple tool calls in parallel.
func (pa *PrimaryAgent) executeToolCalls(ctx context.Context, toolCalls []providerpkg.ToolCall) []ToolExecutionResult {
	results := make([]ToolExecutionResult, len(toolCalls))
	var wg sync.WaitGroup

	for i, tc := range toolCalls {
		wg.Add(1)
		go func(idx int, call providerpkg.ToolCall) {
			defer wg.Done()

			result, err := tools.InvokeTool(ctx, call.Name, call.Arguments)
			results[idx] = ToolExecutionResult{
				ToolCallID: call.ID,
				ToolName:   call.Name,
				Result:     result,
				Error:      err,
			}
		}(i, tc)
	}

	wg.Wait()
	return results
}

// toolResultMaxChars caps tool result content to this many characters before
// injecting it into the message history. Long file reads or command outputs
// grow the context window fast — at 4 chars/token this is ~1K tokens per result.
const toolResultMaxChars = 4096

// formatToolResults formats tool execution results as JSON strings for the model.
// Results are truncated to toolResultMaxChars to prevent unbounded context growth
// across multiple tool-calling iterations.
func formatToolResults(results []ToolExecutionResult) []providerpkg.Message {
	messages := make([]providerpkg.Message, len(results))

	for i, result := range results {
		var content string
		if result.Error != nil {
			content = fmt.Sprintf(`{"error": "%s"}`, result.Error.Error())
		} else {
			jsonData, err := json.Marshal(result.Result)
			if err != nil {
				content = fmt.Sprintf(`{"error": "failed to marshal result: %s"}`, err.Error())
			} else {
				content = string(jsonData)
			}
		}

		// Truncate to prevent runaway context growth across iterations.
		if len(content) > toolResultMaxChars {
			content = content[:toolResultMaxChars] + `... [truncated]`
		}

		messages[i] = providerpkg.Message{
			Role:       "tool",
			Content:    content,
			ToolCallID: result.ToolCallID,
		}
	}

	return messages
}

// HandleQueryWithTools processes a query with full tool-calling support.
// This is the main method that orchestrates intent classification, tool execution,
// and iterative model interactions.
func (pa *PrimaryAgent) HandleQueryWithTools(ctx context.Context, req QueryRequest) (*Response, error) {
	var rootSpan *trace.Span
	var traceID string

	if pa.traceLogger != nil {
		sessionID := req.UserID
		if sessionID == "" {
			sessionID = "default"
		}

		tid, err := pa.traceLogger.StartTrace(ctx, sessionID)
		if err != nil {
			return nil, fmt.Errorf("failed to start trace: %w", err)
		}
		traceID = tid

		rootSpan, err = pa.traceLogger.StartSpan(ctx, traceID, "HandleQueryWithTools", map[string]interface{}{
			"query":     req.Query,
			"user_id":   req.UserID,
			"user_tier": req.UserTier,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to start root span: %w", err)
		}
		ctx = trace.WithSpan(ctx, rootSpan)

		defer func() {
			if rootSpan != nil {
				if err := pa.traceLogger.EndTrace(ctx, traceID, "completed"); err == nil {
					if err := pa.traceLogger.EndSpan(ctx, rootSpan, nil); err != nil {
						slog.Warn("trace: EndSpan failed", "error", err)
					}
				}
			}
		}()
	}

	// Step 1: Fast intent classification via Mini Agent
	var intent Intent
	var confidence float64

	if pa.traceLogger != nil {
		classifySpan, err := pa.traceLogger.StartSpan(ctx, traceID, "intent_classification", map[string]interface{}{
			"query": req.Query,
		})
		if err == nil {
			ctx = trace.WithSpan(ctx, classifySpan)
			intent, confidence = pa.miniAgent.ClassifyIntent(ctx, req.Query)
			if err := pa.traceLogger.EndSpan(ctx, classifySpan, map[string]interface{}{
				"intent":           string(intent),
				"confidence":       confidence,
				"confidence_level": getConfidenceLevel(confidence),
			}); err != nil {
				slog.Warn("trace: EndSpan failed", "error", err)
			}
			ctx = trace.WithSpan(ctx, rootSpan)
		}
	} else {
		intent, confidence = pa.miniAgent.ClassifyIntent(ctx, req.Query)
	}

	// v0.0.30: Check if query should use orchestration
	// Route complex multi-step queries to orchestration engine if enabled
	if pa.useOrchestration && pa.shouldUseOrchestration(intent, confidence, req.Query, req.UserTier) {
		if pa.traceLogger != nil && rootSpan != nil {
			rootSpan.AddMetadata("routing_decision", "orchestration")
			rootSpan.AddMetadata("orchestration_reason", pa.getOrchestrationReason(intent, confidence, req.Query))
			if err := pa.traceLogger.EndSpan(ctx, rootSpan, nil); err != nil {
				slog.Warn("trace: EndSpan failed", "error", err)
			}
		}
		return pa.HandleQueryWithOrchestration(ctx, req)
	}

	// Step 2: Select appropriate provider based on intent and user tier
	var providerName string

	if pa.traceLogger != nil {
		providerSpan, err := pa.traceLogger.StartSpan(ctx, traceID, "provider_selection", map[string]interface{}{
			"intent":             intent,
			"user_tier":          req.UserTier,
			"requested_provider": req.ProviderName,
		})
		if err == nil {
			ctx = trace.WithSpan(ctx, providerSpan)

			providerName = req.ProviderName
			if providerName == "" {
				providerName = pa.selectProvider(req.UserTier, intent)
			}

			pa.traceLogger.EndSpan(ctx, providerSpan, map[string]interface{}{
				"selected_provider": providerName,
			})
			ctx = trace.WithSpan(ctx, rootSpan)
		}
	} else {
		providerName = req.ProviderName
		if providerName == "" {
			providerName = pa.selectProvider(req.UserTier, intent)
		}
	}

	// Step 3: Get model provider plugin
	provider, err := pa.pluginManager.GetProvider(providerName)
	if err != nil {
		if pa.traceLogger != nil && rootSpan != nil {
			if err := pa.traceLogger.FailSpan(ctx, rootSpan, fmt.Sprintf("failed to get provider: %v", err)); err != nil {
				slog.Warn("trace: FailSpan failed", "error", err)
			}
		}
		return nil, fmt.Errorf("failed to get provider %s: %w", providerName, err)
	}

	// Step 4: Get available models
	ctxWithTimeout, cancel := context.WithTimeout(ctx, pa.timeout)
	defer cancel()

	models, err := provider.ListModels(ctxWithTimeout)
	if err != nil {
		return nil, fmt.Errorf("failed to list models: %w", err)
	}

	if len(models) == 0 {
		return nil, fmt.Errorf("provider %s has no available models", providerName)
	}

	// Select model
	modelID := req.ModelID
	if modelID == "" {
		modelID = models[0].ID
	} else {
		modelFound := false
		for _, m := range models {
			if m.ID == modelID {
				modelFound = true
				break
			}
		}
		if !modelFound {
			return nil, fmt.Errorf("model %s not found in provider %s", modelID, providerName)
		}
	}

	// Step 5: Build messages with intent-based system prompt and conversation history
	systemPrompt := pa.buildSystemPrompt(intent)
	if req.SystemPromptOverride != "" {
		systemPrompt = req.SystemPromptOverride
	}
	// Append workspace context so the model uses correct paths with file tools.
	if req.WorkspaceRoot != "" {
		systemPrompt += fmt.Sprintf("\n\nWorkspace root: %s\nWhen using file tools, use paths relative to this workspace root or absolute paths.", req.WorkspaceRoot)
	}

	var messages []providerpkg.Message

	if pa.traceLogger != nil {
		contextSpan, err := pa.traceLogger.StartSpan(ctx, traceID, "context_building", map[string]interface{}{
			"use_memory": req.UseMemory,
		})
		if err == nil {
			ctx = trace.WithSpan(ctx, contextSpan)

			var tiersUsed map[memory.ContextTier]int
			messages, tiersUsed, err = pa.buildContextWithGarden(ctxWithTimeout, systemPrompt, req.Query, req.UserID, req.UseMemory)
			if err != nil {
				if err := pa.traceLogger.FailSpan(ctx, contextSpan, fmt.Sprintf("failed to build context: %v", err)); err != nil {
					slog.Warn("trace: FailSpan failed", "error", err)
				}
				ctx = trace.WithSpan(ctx, rootSpan)
				return nil, err
			}

			spanMetadata := map[string]interface{}{
				"message_count": len(messages),
			}
			if tiersUsed != nil {
				spanMetadata["tiers_used"] = tiersUsed
			}
			pa.traceLogger.EndSpan(ctx, contextSpan, spanMetadata)

			ctx = trace.WithSpan(ctx, rootSpan)
		}
	} else {
		var err error
		messages, _, err = pa.buildContextWithGarden(ctxWithTimeout, systemPrompt, req.Query, req.UserID, req.UseMemory)
		if err != nil {
			return nil, err
		}
	}

	// v0.0.18: Add project_id to context for thread-safe project scoping
	// Using context.Context ensures concurrent requests don't interfere with each other
	if req.ProjectID != "" {
		ctxWithTimeout = tools.WithProjectID(ctxWithTimeout, req.ProjectID)
	}
	// Inject workspace root so file tools can resolve relative paths correctly.
	// The CLI sends os.Getwd() from the user's session; without it the gateway
	// would resolve relative paths against its own process CWD.
	if req.WorkspaceRoot != "" {
		ctxWithTimeout = tools.WithWorkspaceRoot(ctxWithTimeout, req.WorkspaceRoot)
	}

	// Step 6: Get available tools filtered by intent.
	// Sending only relevant tools reduces input token count per call, which is
	// critical for staying under provider TPM rate limits. The full tool set (~33)
	// is only sent when intent is unrecognised.
	availableTools := tools.GetToolsForIntent(string(intent))
	pluginTools := convertToolsToPluginFormat(availableTools)

	// Set defaults for temperature and max tokens
	temperature := req.Temperature
	if temperature == 0 {
		temperature = DefaultTemperature
	}
	maxTokens := req.MaxTokens
	if maxTokens == 0 {
		maxTokens = DefaultMaxTokens
	}

	// Step 7: Iterative tool calling loop
	var finalResponse *providerpkg.CompletionResponse
	var allToolCalls []providerpkg.ToolCall
	var allToolResults []ToolExecutionResult

	for iteration := 0; iteration < pa.maxToolIterations; iteration++ {
		// On the first iteration, force a tool call so the model acts rather than
		// just writing a plan. On subsequent iterations, let it decide — it may
		// want to return a final text response once tool results are available.
		toolChoice := "auto"
		if iteration == 0 && len(pluginTools) > 0 {
			toolChoice = "required"
		}

		completionReq := &providerpkg.CompletionRequest{
			Model:       modelID,
			Messages:    messages,
			Temperature: temperature,
			MaxTokens:   maxTokens,
			Tools:       pluginTools,
			Stream:      false,
			ToolChoice:  toolChoice,
		}

		var response *providerpkg.CompletionResponse

		if pa.traceLogger != nil {
			modelSpan, err := pa.traceLogger.StartSpan(ctx, traceID, "model_invocation", map[string]interface{}{
				"model":     modelID,
				"provider":  providerName,
				"iteration": iteration,
			})
			if err == nil {
				ctx = trace.WithSpan(ctx, modelSpan)
				response, err = provider.GenerateCompletion(ctxWithTimeout, completionReq)
				if err != nil {
					if err := pa.traceLogger.FailSpan(ctx, modelSpan, fmt.Sprintf("completion failed: %v", err)); err != nil {
						slog.Warn("trace: FailSpan failed", "error", err)
					}
					ctx = trace.WithSpan(ctx, rootSpan)
					slog.Error("completion failed",
						"component", "agent_runner",
						"method", "HandleQueryWithTools",
						"error", err,
						"query_preview", truncateQuery(req.Query, 200),
						"model", modelID,
						"provider", providerName,
						"iteration", iteration,
					)
					return nil, fmt.Errorf("completion failed: %w", err)
				}

				pa.traceLogger.EndSpan(ctx, modelSpan, map[string]interface{}{
					"input_tokens":  response.Usage.InputTokens,
					"output_tokens": response.Usage.OutputTokens,
					"tool_calls":    len(response.ToolCalls),
				})
				ctx = trace.WithSpan(ctx, rootSpan)
			}
		} else {
			var err error
			response, err = provider.GenerateCompletion(ctxWithTimeout, completionReq)
			if err != nil {
				slog.Error("completion failed",
					"component", "agent_runner",
					"method", "HandleQueryWithTools",
					"error", err,
					"query_preview", truncateQuery(req.Query, 200),
					"model", modelID,
					"provider", providerName,
					"iteration", iteration,
				)
				return nil, fmt.Errorf("completion failed: %w", err)
			}
		}

		// If no tool calls, we're done
		if len(response.ToolCalls) == 0 {
			finalResponse = response
			break
		}

		// Execute tool calls in parallel
		allToolCalls = append(allToolCalls, response.ToolCalls...)

		var toolResults []ToolExecutionResult

		// v0.0.17: Trace tool execution
		if pa.traceLogger != nil {
			toolExecSpan, err := pa.traceLogger.StartSpan(ctx, traceID, "tool_execution", map[string]interface{}{
				"tool_call_count": len(response.ToolCalls),
			})
			if err == nil {
				ctx = trace.WithSpan(ctx, toolExecSpan)
				toolResults = pa.executeToolCalls(ctxWithTimeout, response.ToolCalls)

				successCount := 0
				for _, result := range toolResults {
					if result.Error == nil {
						successCount++
					}
				}

				pa.traceLogger.EndSpan(ctx, toolExecSpan, map[string]interface{}{
					"success_count": successCount,
					"total_count":   len(toolResults),
				})
				ctx = trace.WithSpan(ctx, rootSpan)
			}
		} else {
			toolResults = pa.executeToolCalls(ctxWithTimeout, response.ToolCalls)
		}

		// v0.0.18: Accumulate tool results
		allToolResults = append(allToolResults, toolResults...)

		// Add assistant message with tool calls.
		// ReasoningContent is preserved so providers that emit it (e.g. Kimi with
		// enable_thinking: false still injecting reasoning_content) don't 400 on
		// the next iteration due to a missing reasoning_content field.
		messages = append(messages, providerpkg.Message{
			Role:             "assistant",
			Content:          response.Content,
			ToolCalls:        response.ToolCalls,
			ReasoningContent: response.ReasoningContent,
		})

		// Add tool results as messages
		toolMessages := formatToolResults(toolResults)
		messages = append(messages, toolMessages...)

		// If this is the last iteration, get final response without tools
		if iteration == pa.maxToolIterations-1 {
			completionReq.Tools = []providerpkg.Tool{}

			if pa.traceLogger != nil {
				finalSpan, err := pa.traceLogger.StartSpan(ctx, traceID, "final_model_invocation", map[string]interface{}{
					"model":    modelID,
					"provider": providerName,
				})
				if err == nil {
					ctx = trace.WithSpan(ctx, finalSpan)
					finalResponse, err = provider.GenerateCompletion(ctxWithTimeout, completionReq)
					if err != nil {
						pa.traceLogger.FailSpan(ctx, finalSpan, fmt.Sprintf("final completion failed: %v", err))
						ctx = trace.WithSpan(ctx, rootSpan)
						return nil, fmt.Errorf("final completion failed: %w", err)
					}

					pa.traceLogger.EndSpan(ctx, finalSpan, map[string]interface{}{
						"input_tokens":  finalResponse.Usage.InputTokens,
						"output_tokens": finalResponse.Usage.OutputTokens,
					})
					ctx = trace.WithSpan(ctx, rootSpan)
				}
			} else {
				var err error
				finalResponse, err = provider.GenerateCompletion(ctxWithTimeout, completionReq)
				if err != nil {
					return nil, fmt.Errorf("final completion failed: %w", err)
				}
			}
			break
		}
	}

	if finalResponse == nil {
		if pa.traceLogger != nil && rootSpan != nil {
			pa.traceLogger.FailSpan(ctx, rootSpan, "no response generated")
		}
		return nil, fmt.Errorf("no response generated after %d iterations", pa.maxToolIterations)
	}

	response := &Response{
		ID:          finalResponse.ID,
		Content:     finalResponse.Content,
		Model:       finalResponse.Model,
		Provider:    providerName,
		Usage:       finalResponse.Usage,
		ToolCalls:   allToolCalls,
		ToolResults: allToolResults,
		Timestamp:   time.Now(),
	}

	if req.UseMemory {
		if pa.traceLogger != nil {
			memorySpan, err := pa.traceLogger.StartSpan(ctx, traceID, "memory_storage", map[string]interface{}{
				"user_id": req.UserID,
			})
			if err == nil {
				ctx = trace.WithSpan(ctx, memorySpan)
				if err := pa.storeConversation(ctxWithTimeout, req.UserID, req.Query, response); err != nil {
					pa.traceLogger.FailSpan(ctx, memorySpan, fmt.Sprintf("failed to store conversation: %v", err))
					ctx = trace.WithSpan(ctx, rootSpan)
					return response, fmt.Errorf("warning: failed to store conversation: %w", err)
				}
				pa.traceLogger.EndSpan(ctx, memorySpan, nil)
				ctx = trace.WithSpan(ctx, rootSpan)
			}
		} else {
			if err := pa.storeConversation(ctxWithTimeout, req.UserID, req.Query, response); err != nil {
				return response, fmt.Errorf("warning: failed to store conversation: %w", err)
			}
		}
	}

	if pa.traceLogger != nil && rootSpan != nil {
		rootSpan.AddMetadata("total_tool_calls", len(allToolCalls))
		rootSpan.AddMetadata("total_tokens", response.Usage.InputTokens+response.Usage.OutputTokens)
	}

	return response, nil
}
