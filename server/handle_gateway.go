package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/DojoGenesis/gateway/disposition"
	"github.com/DojoGenesis/gateway/events"
	orchestrationpkg "github.com/DojoGenesis/gateway/orchestration"
	"github.com/DojoGenesis/gateway/pkg/collaboration"
	pkgerrors "github.com/DojoGenesis/gateway/pkg/errors"
	"github.com/DojoGenesis/gateway/pkg/gateway"
	"github.com/DojoGenesis/gateway/pkg/intelligence"
	"github.com/DojoGenesis/gateway/pkg/reflection"
	"github.com/DojoGenesis/gateway/pkg/validation"
	"github.com/DojoGenesis/gateway/provider"
	"github.com/DojoGenesis/gateway/server/services/providers"
	"github.com/DojoGenesis/gateway/server/streaming"
)

// ─── Agent chat types ────────────────────────────────────────────────────────

// PatchIntent represents a structured document edit proposal embedded in an
// agent response. The agent emits a fenced JSON block; parsePatchIntent extracts
// it and stripPatchIntent removes it from the visible response text.
type PatchIntent struct {
	Operation   string  `json:"operation"`             // "replace" | "insert" | "append"
	SectionID   *string `json:"section_id,omitempty"`  // target section; nil = whole document
	Content     string  `json:"content"`               // the new text to apply
	Description string  `json:"description,omitempty"` // human-readable summary shown in UI
}

// DocumentContext carries optional document data the caller passes into the chat
// request so the agent can ground its responses on the current document state.
type DocumentContext struct {
	ID      string `json:"id"`
	Content string `json:"content"`
}

// patchIntentPattern matches a fenced JSON block the agent uses to signal edits:
//
//	```patch_intent
//	{ … }
//	```
var patchIntentPattern = regexp.MustCompile("(?s)```patch_intent\\s*(\\{.*?\\})\\s*```")

// parsePatchIntent scans the agent's raw response text for an embedded
// patch_intent block and, if found, deserialises it into a PatchIntent.
// Returns nil when no block is present; never returns an error (malformed
// blocks are silently ignored so the chat still succeeds).
func parsePatchIntent(text string) *PatchIntent {
	m := patchIntentPattern.FindStringSubmatch(text)
	if len(m) < 2 {
		return nil
	}
	var pi PatchIntent
	if err := json.Unmarshal([]byte(m[1]), &pi); err != nil {
		return nil
	}
	if pi.Operation == "" || pi.Content == "" {
		return nil
	}
	return &pi
}

// stripPatchIntent removes the raw patch_intent fenced block from the agent
// response so the frontend only displays the human-readable text.
func stripPatchIntent(text string) string {
	cleaned := patchIntentPattern.ReplaceAllString(text, "")
	return strings.TrimSpace(cleaned)
}

// ─── runAgentLoop ─────────────────────────────────────────────────────────────

// runAgentLoop drives the LLM ↔ tool agentic loop:
//  1. Call GenerateCompletion with the accumulated messages + available tools.
//  2. If the LLM returns tool calls, execute each one via the gateway ToolRegistry.
//  3. Append tool results as tool-role messages and repeat.
//  4. Stop when no more tool calls are requested or maxIter is reached.
//
// Returns the final CompletionResponse (last LLM turn without tool calls).
func (s *Server) runAgentLoop(
	ctx context.Context,
	llmProvider provider.ModelProvider,
	messages []provider.Message,
	tools []provider.Tool,
) (*provider.CompletionResponse, error) {
	const maxIter = 8

	for i := 0; i < maxIter; i++ {
		req := &provider.CompletionRequest{
			Messages: messages,
			Tools:    tools,
		}

		resp, err := llmProvider.GenerateCompletion(ctx, req)
		if err != nil {
			return nil, err
		}

		// No tool calls → we have the final answer
		if len(resp.ToolCalls) == 0 {
			return resp, nil
		}

		// Append the assistant turn (with tool calls) to history
		assistantMsg := provider.Message{
			Role:      "assistant",
			Content:   resp.Content,
			ToolCalls: resp.ToolCalls,
		}
		messages = append(messages, assistantMsg)

		// Execute each tool call and collect results
		for _, tc := range resp.ToolCalls {
			result := s.executeToolCall(ctx, tc)
			toolMsg := provider.Message{
				Role:       "tool",
				Content:    result,
				ToolCallID: tc.ID,
			}
			messages = append(messages, toolMsg)
		}
	}

	// maxIter reached — do one final completion without tools to get a response
	finalResp, err := llmProvider.GenerateCompletion(ctx, &provider.CompletionRequest{
		Messages: messages,
	})
	if err != nil {
		return nil, err
	}
	return finalResp, nil
}

// runAgentLoopWithDisposition drives the LLM ↔ tool agentic loop with
// disposition-aware error handling and collaboration check-ins.
// Falls back to runAgentLoop when runtime is nil.
func (s *Server) runAgentLoopWithDisposition(
	ctx context.Context,
	llmProvider provider.ModelProvider,
	messages []provider.Message,
	tools []provider.Tool,
	runtime *AgentRuntime,
) (*provider.CompletionResponse, error) {
	if runtime == nil {
		return s.runAgentLoop(ctx, llmProvider, messages, tools)
	}

	const maxIter = 8

	for i := 0; i < maxIter; i++ {
		req := &provider.CompletionRequest{
			Messages: messages,
			Tools:    tools,
		}

		resp, err := llmProvider.GenerateCompletion(ctx, req)
		if err != nil {
			// Use disposition-aware error handling
			if runtime.ErrorHandler != nil {
				decision := runtime.ErrorHandler.HandleError(ctx, err, i)
				if decision.ShouldRetry() {
					continue
				}
				if decision.ShouldContinue() {
					// Return what we have so far
					return &provider.CompletionResponse{
						Content: fmt.Sprintf("Encountered an error but continuing: %s", decision.Message),
					}, nil
				}
			}
			return nil, err
		}

		// No tool calls → we have the final answer
		if len(resp.ToolCalls) == 0 {
			return resp, nil
		}

		// Append the assistant turn (with tool calls) to history
		assistantMsg := provider.Message{
			Role:      "assistant",
			Content:   resp.Content,
			ToolCalls: resp.ToolCalls,
		}
		messages = append(messages, assistantMsg)

		// Execute each tool call and collect results
		for _, tc := range resp.ToolCalls {
			result := s.executeToolCall(ctx, tc)
			toolMsg := provider.Message{
				Role:       "tool",
				Content:    result,
				ToolCallID: tc.ID,
			}
			messages = append(messages, toolMsg)
		}

		// Check if collaboration manager says we should check in
		if runtime.CollabManager != nil {
			event := collaboration.CollabEvent{
				Type:          "action",
				IsSignificant: len(resp.ToolCalls) > 1,
				Description:   fmt.Sprintf("Executed %d tool calls", len(resp.ToolCalls)),
			}
			if runtime.CollabManager.ShouldCheckIn(ctx, event) {
				// Inject a check-in prompt into the conversation
				messages = append(messages, provider.Message{
					Role:    "system",
					Content: "Pause and check in with the user about your progress before continuing.",
				})
			}
		}
	}

	// maxIter reached — do one final completion without tools to get a response
	finalResp, err := llmProvider.GenerateCompletion(ctx, &provider.CompletionRequest{
		Messages: messages,
	})
	if err != nil {
		return nil, err
	}
	return finalResp, nil
}

// executeToolCall dispatches a single tool call through the gateway ToolRegistry.
// Returns a JSON-encoded result string (or an error description) suitable for
// inclusion in a tool-role message.
func (s *Server) executeToolCall(ctx context.Context, tc provider.ToolCall) string {
	if s.toolRegistry == nil {
		return `{"error":"tool registry not available"}`
	}

	toolDef, err := s.toolRegistry.Get(ctx, tc.Name)
	if err != nil {
		return fmt.Sprintf(`{"error":"tool not found: %s"}`, tc.Name)
	}

	toolCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	result, err := toolDef.Function(toolCtx, tc.Arguments)
	if err != nil {
		return fmt.Sprintf(`{"error":"%s"}`, err.Error())
	}

	data, err := json.Marshal(result)
	if err != nil {
		return `{"error":"failed to marshal tool result"}`
	}
	return string(data)
}

// buildToolList converts ToolRegistry entries into the provider.Tool slice
// required by CompletionRequest.
func (s *Server) buildToolList(ctx context.Context) []provider.Tool {
	if s.toolRegistry == nil {
		return nil
	}
	defs, err := s.toolRegistry.List(ctx)
	if err != nil || len(defs) == 0 {
		return nil
	}
	tools := make([]provider.Tool, 0, len(defs))
	for _, d := range defs {
		tools = append(tools, provider.Tool{
			Name:        d.Name,
			Description: d.Description,
			Parameters:  d.Parameters,
		})
	}
	return tools
}

// ─── v1.2 Streaming Agent Chat ──────────────────────────────────────────────

// agentChatRequest is the shared request type for both streaming and
// non-streaming agent chat. The Stream field determines the response format.
type agentChatRequest struct {
	Message         string `json:"message" binding:"required"`
	UserID          string `json:"user_id"`
	Model           string `json:"model,omitempty"`
	Provider        string `json:"provider,omitempty"`
	DocumentID      string `json:"document_id,omitempty"`
	DocumentContent string `json:"document_content,omitempty"`
	Stream          bool   `json:"stream"`
}

// buildAgentSystemPrompt constructs the system prompt with disposition context
// and optional document context. Used by both streaming and non-streaming paths.
func (s *Server) buildAgentSystemPrompt(agentConfig *gateway.AgentConfig, documentID, documentContent string) string {
	systemPrompt := fmt.Sprintf(
		"You are a helpful scientific writing assistant.\n"+
			"Disposition: Pacing=%s, Depth=%s, Tone=%s, Initiative=%s.\n"+
			"Adopt a %s communication style. "+
			"Respond concisely and helpfully to the user's message.\n\n"+
			"When you want to propose a specific edit to the document, embed EXACTLY ONE patch_intent "+
			"fenced code block in your response. The block MUST use this format (no extra keys, no markdown inside the JSON):\n\n"+
			"```patch_intent\n"+
			"{\"operation\":\"replace\",\"section_id\":\"<section id or null>\",\"content\":\"<new text>\",\"description\":\"<short summary>\"}\n"+
			"```\n\n"+
			"Supported operations: replace, insert, append.\n\n"+
			"EXAMPLE — user asks to fix a typo in the introduction:\n\n"+
			"Sure, I found the typo in the second sentence. Here is the corrected version:\n\n"+
			"```patch_intent\n"+
			"{\"operation\":\"replace\",\"section_id\":\"intro-1\",\"content\":\"The experiment confirmed the hypothesis.\",\"description\":\"Fix typo: confrimed → confirmed\"}\n"+
			"```\n\n"+
			"Place the block at the end of your response. Do NOT wrap it in another code fence.",
		agentConfig.Pacing, agentConfig.Depth, agentConfig.Tone, agentConfig.Initiative,
		agentConfig.Tone,
	)

	if documentContent != "" {
		systemPrompt += fmt.Sprintf(
			"\n\n--- CURRENT DOCUMENT (id: %s) ---\n%s\n--- END DOCUMENT ---",
			documentID, documentContent,
		)
	}
	return systemPrompt
}

// writeSSEEvent marshals a StreamEvent to SSE wire format and flushes.
func (s *Server) writeSSEEvent(w http.ResponseWriter, flusher http.Flusher, evt events.StreamEvent) {
	data, _ := json.Marshal(evt)
	fmt.Fprintf(w, "event: %s\ndata: %s\n\n", evt.Type, data)
	flusher.Flush()
	if s.telemetryTap != nil {
		s.telemetryTap.Push(evt)
	}
}

// handleGatewayAgentChatStream handles the SSE streaming variant of agent chat.
// Invoked when the request includes "stream": true.
func (s *Server) handleGatewayAgentChatStream(c *gin.Context, agentID string, req *agentChatRequest, agentConfig *gateway.AgentConfig) {
	eventChan := make(chan events.StreamEvent, 100)

	// SSE headers
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		s.errorResponse(c, http.StatusInternalServerError, "server_error", "Streaming not supported")
		return
	}

	var llmProvider provider.ModelProvider
	var err error
	if req.Provider != "" {
		if prov, ok := s.pluginManager.GetProviders()[req.Provider]; ok {
			llmProvider = prov
		}
	}
	if llmProvider == nil {
		llmProvider, err = s.resolveProvider(req.Model)
		if err != nil {
			s.writeSSEEvent(c.Writer, flusher, events.NewErrorEvent(err.Error(), "PROVIDER_ERROR"))
			fmt.Fprintf(c.Writer, "data: [DONE]\n\n")
			flusher.Flush()
			return
		}
	}

	systemPrompt := s.buildAgentSystemPrompt(agentConfig, req.DocumentID, req.DocumentContent)
	messages := []provider.Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: req.Message},
	}
	toolList := s.buildToolList(c.Request.Context())

	llmCtx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)

	// Launch the streaming agent loop in a goroutine
	go func() {
		defer cancel()
		s.runAgentLoopStreaming(llmCtx, agentID, llmProvider, messages, toolList, eventChan)
	}()

	// Drain the event channel into SSE
	for evt := range eventChan {
		s.writeSSEEvent(c.Writer, flusher, evt)
	}

	// Terminator
	fmt.Fprintf(c.Writer, "data: [DONE]\n\n")
	flusher.Flush()
}

// runAgentLoopStreaming drives the LLM ↔ tool agentic loop while emitting
// SSE events for each phase: thinking, tool invocations, response chunks,
// patch_intent, and the final complete event.
func (s *Server) runAgentLoopStreaming(
	ctx context.Context,
	agentID string,
	llmProvider provider.ModelProvider,
	messages []provider.Message,
	tools []provider.Tool,
	eventChan chan<- events.StreamEvent,
) {
	defer close(eventChan)
	startTime := time.Now()
	const maxIter = 8

	// Thinking indicator
	eventChan <- events.NewThinkingEvent("Processing your request...")

	var finalResp *provider.CompletionResponse

	for i := 0; i < maxIter; i++ {
		resp, err := llmProvider.GenerateCompletion(ctx, &provider.CompletionRequest{
			Messages: messages,
			Tools:    tools,
		})
		if err != nil {
			eventChan <- events.NewErrorEvent(err.Error(), "LLM_ERROR")
			return
		}

		if len(resp.ToolCalls) == 0 {
			finalResp = resp
			break
		}

		// Append the assistant turn (with tool calls) to history
		assistantMsg := provider.Message{
			Role:      "assistant",
			Content:   resp.Content,
			ToolCalls: resp.ToolCalls,
		}
		messages = append(messages, assistantMsg)

		// Execute each tool call and emit events
		for _, tc := range resp.ToolCalls {
			eventChan <- events.NewToolInvokedEvent(tc.Name, tc.Arguments)
			tcStart := time.Now()
			result := s.executeToolCall(ctx, tc)
			durationMs := time.Since(tcStart).Milliseconds()
			eventChan <- events.NewToolCompletedEvent(tc.Name, result, durationMs)

			messages = append(messages, provider.Message{
				Role:       "tool",
				Content:    result,
				ToolCallID: tc.ID,
			})
		}
	}

	if finalResp == nil {
		// maxIter reached — one final completion without tools
		resp, err := llmProvider.GenerateCompletion(ctx, &provider.CompletionRequest{
			Messages: messages,
		})
		if err != nil {
			eventChan <- events.NewErrorEvent(err.Error(), "LLM_ERROR")
			return
		}
		finalResp = resp
	}

	rawText := finalResp.Content
	if rawText == "" {
		rawText = "I received your message but couldn't generate a response."
	}

	// Parse patch_intent from the complete response, then stream the clean text
	patchIntent := parsePatchIntent(rawText)
	cleanText := stripPatchIntent(rawText)

	chunks := streaming.SplitIntoChunks(cleanText, 50)
	for _, chunk := range chunks {
		eventChan <- events.NewResponseChunkEvent(chunk)
		time.Sleep(50 * time.Millisecond)
	}

	// Emit patch_intent as a dedicated event (if present)
	if patchIntent != nil {
		eventChan <- events.NewPatchIntentEvent(
			patchIntent.Operation,
			patchIntent.SectionID,
			patchIntent.Content,
			patchIntent.Description,
		)
	}

	// Complete event with usage stats
	elapsed := time.Since(startTime).Milliseconds()
	eventChan <- events.NewCompleteEvent(map[string]interface{}{
		"agent_id":      agentID,
		"input_tokens":  finalResp.Usage.InputTokens,
		"output_tokens": finalResp.Usage.OutputTokens,
		"total_tokens":  finalResp.Usage.TotalTokens,
		"duration_ms":   elapsed,
	})
}

// handleGatewayListTools returns all registered tools with namespace info and MCP server origin.
// GET /v1/gateway/tools
func (s *Server) handleGatewayListTools(c *gin.Context) {
	if s.toolRegistry == nil {
		s.errorResponse(c, http.StatusServiceUnavailable, "service_unavailable", "Tool registry not available")
		return
	}

	tools, err := s.toolRegistry.List(c.Request.Context())
	if err != nil {
		s.errorResponse(c, http.StatusInternalServerError, "server_error", fmt.Sprintf("Failed to list tools: %v", err))
		return
	}

	// Transform to API response format
	toolsResp := make([]map[string]interface{}, 0, len(tools))
	for _, tool := range tools {
		toolsResp = append(toolsResp, map[string]interface{}{
			"name":        tool.Name,
			"description": tool.Description,
			"parameters":  tool.Parameters,
			"namespace":   extractNamespace(tool.Name),
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"tools": toolsResp,
		"count": len(toolsResp),
	})
}

// handleGatewayCreateAgent creates a new agent with the provided disposition configuration.
// POST /v1/gateway/agents
func (s *Server) handleGatewayCreateAgent(c *gin.Context) {
	var req struct {
		WorkspaceRoot string `json:"workspace_root" binding:"required"`
		ActiveMode    string `json:"active_mode"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		s.errorResponse(c, http.StatusBadRequest, "invalid_request", fmt.Sprintf("Invalid request: %v", err))
		return
	}

	if s.agentInitializer == nil {
		s.errorResponse(c, http.StatusServiceUnavailable, "service_unavailable", "Agent initializer not available")
		return
	}

	// Initialize agent configuration from disposition
	agentConfig, err := s.agentInitializer.Initialize(c.Request.Context(), req.WorkspaceRoot, req.ActiveMode)
	if err != nil {
		s.errorResponse(c, http.StatusInternalServerError, "agent_init_failed", fmt.Sprintf("Failed to initialize agent: %v", err))
		return
	}

	// Load full ADA disposition for consumer modules
	disp, dispErr := disposition.ResolveDisposition(req.WorkspaceRoot, req.ActiveMode)
	if dispErr != nil {
		disp = disposition.DefaultDisposition()
	}

	// Generate agent ID
	agentID := uuid.New().String()

	// Instantiate per-agent consumer modules with disposition
	runtime := &AgentRuntime{
		Config:        agentConfig,
		Disposition:   disp,
		ErrorHandler:  pkgerrors.NewHandler(pkgerrors.WithDisposition(disp)),
		CollabManager: collaboration.NewManager(collaboration.WithDisposition(disp)),
		Validator:     validation.NewValidator(validation.WithDisposition(disp)),
		Reflection:    reflection.NewEngine(reflection.WithDisposition(disp)),
		Proactive:     intelligence.NewProactiveEngine(intelligence.WithDisposition(disp)),
	}

	// Store agent runtime (in-memory for now - production would use persistence)
	s.agentMu.Lock()
	s.agents[agentID] = runtime
	s.agentMu.Unlock()

	c.JSON(http.StatusCreated, gin.H{
		"agent_id": agentID,
		"config":   agentConfig,
		"disposition": gin.H{
			"pacing":     disp.Pacing,
			"depth":      disp.Depth,
			"tone":       disp.Tone,
			"initiative": disp.Initiative,
			"source":     disp.SourceFile,
		},
	})
}

// handleGatewayGetAgent retrieves agent status and current disposition.
// GET /v1/gateway/agents/:id
func (s *Server) handleGatewayGetAgent(c *gin.Context) {
	agentID := c.Param("id")

	s.agentMu.RLock()
	runtime, exists := s.agents[agentID]
	s.agentMu.RUnlock()

	if !exists {
		s.errorResponse(c, http.StatusNotFound, "not_found", fmt.Sprintf("Agent not found: %s", agentID))
		return
	}

	resp := gin.H{
		"agent_id": agentID,
		"config":   runtime.Config,
		"status":   "active",
	}
	if runtime.Disposition != nil {
		resp["disposition"] = gin.H{
			"pacing":     runtime.Disposition.Pacing,
			"depth":      runtime.Disposition.Depth,
			"tone":       runtime.Disposition.Tone,
			"initiative": runtime.Disposition.Initiative,
		}
	}
	c.JSON(http.StatusOK, resp)
}

// handleGatewayAgentChat handles chat interactions with a specific agent.
// It runs a full LLM ↔ tool agentic loop via runAgentLoop, populates
// patch_intent when the response contains an embedded edit proposal, and
// injects document context into the system prompt when provided.
//
// When the request includes "stream": true, the response is delivered as
// an SSE event stream instead of a single JSON response (v1.2).
//
// POST /v1/gateway/agents/:id/chat
func (s *Server) handleGatewayAgentChat(c *gin.Context) {
	agentID := c.Param("id")

	var req agentChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		s.errorResponse(c, http.StatusBadRequest, "invalid_request", fmt.Sprintf("Invalid request: %v", err))
		return
	}

	s.agentMu.RLock()
	runtime, exists := s.agents[agentID]
	s.agentMu.RUnlock()

	if !exists {
		s.errorResponse(c, http.StatusNotFound, "not_found", fmt.Sprintf("Agent not found: %s", agentID))
		return
	}

	agentConfig := runtime.Config

	if s.pluginManager == nil {
		s.errorResponse(c, http.StatusServiceUnavailable, "service_unavailable", "Plugin manager not available")
		return
	}

	// v1.2: Streaming path — delegate to SSE handler
	if req.Stream {
		s.handleGatewayAgentChatStream(c, agentID, &req, agentConfig)
		return
	}

	// ── Non-streaming path ──────────────────────────────────────────────────

	// Resolve the LLM provider — honour explicit provider/model from request.
	var llmProvider provider.ModelProvider
	var err error
	if req.Provider != "" {
		if prov, ok := s.pluginManager.GetProviders()[req.Provider]; ok {
			llmProvider = prov
		}
	}
	if llmProvider == nil {
		llmProvider, err = s.resolveProvider(req.Model)
		if err != nil {
			s.errorResponse(c, http.StatusServiceUnavailable, "provider_unavailable", fmt.Sprintf("No LLM provider available: %v", err))
			return
		}
	}

	systemPrompt := s.buildAgentSystemPrompt(agentConfig, req.DocumentID, req.DocumentContent)

	// Use a generous context for LLM calls (5 minutes) — Ollama can be slow on first load
	llmCtx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Build tool list from registry (nil-safe; returns nil when no registry)
	toolList := s.buildToolList(llmCtx)

	messages := []provider.Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: req.Message},
	}

	// Run the agentic loop with disposition-aware error handling
	completionResp, err := s.runAgentLoopWithDisposition(llmCtx, llmProvider, messages, toolList, runtime)
	if err != nil {
		s.errorResponseWithDetails(c, http.StatusInternalServerError, "llm_error",
			fmt.Sprintf("LLM completion failed: %v", err),
			gin.H{"agent_id": agentID})
		return
	}

	rawText := completionResp.Content
	if rawText == "" {
		rawText = "I received your message but couldn't generate a response."
	}

	// Extract and strip patch_intent block from the visible response
	patchIntent := parsePatchIntent(rawText)
	responseText := stripPatchIntent(rawText)

	// Build tool_calls summary for the response
	toolCallsSummary := make([]map[string]interface{}, 0, len(completionResp.ToolCalls))
	for _, tc := range completionResp.ToolCalls {
		toolCallsSummary = append(toolCallsSummary, map[string]interface{}{
			"id":   tc.ID,
			"name": tc.Name,
		})
	}

	taskID := uuid.New().String()

	resp := gin.H{
		"agent_id": agentID,
		"response": responseText,
		"task_id":  taskID,
		"message":  req.Message,
		"disposition": map[string]interface{}{
			"pacing":     agentConfig.Pacing,
			"depth":      agentConfig.Depth,
			"tone":       agentConfig.Tone,
			"initiative": agentConfig.Initiative,
		},
		"tool_calls": toolCallsSummary,
	}
	if patchIntent != nil {
		resp["patch_intent"] = patchIntent
	}

	c.JSON(http.StatusOK, resp)
}

// handleGatewayOrchestrate submits an orchestration plan and returns execution ID.
// POST /v1/gateway/orchestrate
func (s *Server) handleGatewayOrchestrate(c *gin.Context) {
	var req struct {
		Plan   gateway.ExecutionPlan `json:"plan" binding:"required"`
		UserID string                `json:"user_id"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		s.errorResponse(c, http.StatusBadRequest, "invalid_request", fmt.Sprintf("Invalid request: %v", err))
		return
	}

	if s.orchestrationEngine == nil {
		s.errorResponse(c, http.StatusServiceUnavailable, "service_unavailable", "Orchestration engine not available")
		return
	}

	// Convert gateway.ExecutionPlan to orchestrationpkg.Plan
	orchPlan := convertToOrchestrationPlan(&req.Plan)

	// Create task
	task := &orchestrationpkg.Task{
		ID:          uuid.New().String(),
		Description: req.Plan.Name,
		CreatedAt:   time.Now(),
	}

	// Register state BEFORE starting the goroutine so the DAG poll endpoint
	// can find it immediately (even while executing). Without this, the DAG
	// handler always returns 404 because s.orchestrations is never populated
	// from this path.
	orchState := &OrchestrationState{
		ID:        task.ID,
		TaskID:    task.ID,
		Status:    "planning",
		CreatedAt: task.CreatedAt,
		Plan:      orchPlan,
	}
	s.orchestrations.Store(orchState)

	// Execute orchestration (async)
	go func() {
		userID := req.UserID
		if userID == "" {
			userID = "anonymous"
		}
		orchState.mu.Lock()
		orchState.Status = "executing"
		orchState.mu.Unlock()

		execErr := s.orchestrationEngine.Execute(c.Request.Context(), orchPlan, task, userID)

		orchState.mu.Lock()
		if execErr != nil {
			orchState.Status = "failed"
			orchState.Error = execErr.Error()
		} else {
			orchState.Status = "complete"
		}
		orchState.mu.Unlock()
	}()

	c.JSON(http.StatusAccepted, gin.H{
		"execution_id": task.ID,
		"plan_id":      req.Plan.ID,
		"status":       "submitted",
	})
}

// handleGatewayOrchestrationDAG retrieves DAG structure and execution status.
// GET /v1/gateway/orchestrate/:id/dag
func (s *Server) handleGatewayOrchestrationDAG(c *gin.Context) {
	executionID := c.Param("id")

	// Retrieve orchestration state from store
	state, exists := s.orchestrations.Get(executionID)
	if !exists {
		s.errorResponse(c, http.StatusNotFound, "not_found", fmt.Sprintf("Orchestration not found: %s", executionID))
		return
	}

	state.mu.Lock()
	status := state.Status
	plan := state.Plan
	state.mu.Unlock()

	if plan == nil {
		c.JSON(http.StatusOK, gin.H{
			"execution_id": executionID,
			"status":       status,
			"dag": map[string]interface{}{
				"nodes": []interface{}{},
				"edges": []interface{}{},
			},
		})
		return
	}

	// Build nodes array with execution status
	nodes := make([]map[string]interface{}, 0, len(plan.Nodes))
	for _, node := range plan.Nodes {
		nodeData := map[string]interface{}{
			"id":        node.ID,
			"tool_name": node.ToolName,
			"state":     string(node.State),
		}

		if node.StartTime != nil {
			nodeData["start_time"] = node.StartTime.Format(time.RFC3339)
		}
		if node.EndTime != nil {
			nodeData["end_time"] = node.EndTime.Format(time.RFC3339)
			nodeData["duration_ms"] = node.EndTime.Sub(*node.StartTime).Milliseconds()
		}
		if node.Error != "" {
			nodeData["error"] = node.Error
		}
		if node.RetryCount > 0 {
			nodeData["retry_count"] = node.RetryCount
		}

		nodes = append(nodes, nodeData)
	}

	// Build edges array from dependencies
	edges := make([]map[string]interface{}, 0)
	for _, node := range plan.Nodes {
		for _, depID := range node.Dependencies {
			edges = append(edges, map[string]interface{}{
				"from": depID,
				"to":   node.ID,
			})
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"execution_id": executionID,
		"status":       status,
		"plan_id":      plan.ID,
		"created_at":   plan.CreatedAt.Format(time.RFC3339),
		"dag": map[string]interface{}{
			"nodes": nodes,
			"edges": edges,
		},
	})
}

// handleGatewayGetTrace retrieves trace details if OTEL is enabled.
// GET /v1/gateway/traces/:id
func (s *Server) handleGatewayGetTrace(c *gin.Context) {
	traceID := c.Param("id")

	if s.traceLogger == nil {
		s.errorResponse(c, http.StatusServiceUnavailable, "service_unavailable", "Trace logger not available")
		return
	}

	// Retrieve trace metadata
	trace, err := s.traceLogger.GetTrace(c.Request.Context(), traceID)
	if err != nil {
		s.errorResponse(c, http.StatusNotFound, "not_found", fmt.Sprintf("Trace not found: %v", err))
		return
	}

	// Retrieve all spans for this trace
	spans, err := s.traceLogger.GetTraceSpans(c.Request.Context(), traceID)
	if err != nil {
		s.errorResponse(c, http.StatusInternalServerError, "server_error", fmt.Sprintf("Failed to retrieve spans: %v", err))
		return
	}

	// Build spans response with relevant details
	spansResp := make([]map[string]interface{}, 0, len(spans))
	for _, span := range spans {
		spanData := map[string]interface{}{
			"span_id":    span.SpanID,
			"parent_id":  span.ParentID,
			"name":       span.Name,
			"start_time": span.StartTime.Format(time.RFC3339),
			"status":     span.Status,
		}

		if span.EndTime != nil {
			spanData["end_time"] = span.EndTime.Format(time.RFC3339)
			spanData["duration_ms"] = span.EndTime.Sub(span.StartTime).Milliseconds()
		}

		if span.Inputs != nil {
			spanData["inputs"] = span.Inputs
		}
		if span.Outputs != nil {
			spanData["outputs"] = span.Outputs
		}
		if span.Metadata != nil {
			spanData["metadata"] = span.Metadata
		}

		spansResp = append(spansResp, spanData)
	}

	// Build response
	response := gin.H{
		"trace_id":   trace.TraceID,
		"session_id": trace.SessionID,
		"start_time": trace.StartTime.Format(time.RFC3339),
		"status":     trace.Status,
		"spans":      spansResp,
		"span_count": len(spansResp),
	}

	if trace.EndTime != nil {
		response["end_time"] = trace.EndTime.Format(time.RFC3339)
		response["duration_ms"] = trace.EndTime.Sub(trace.StartTime).Milliseconds()
	}

	if trace.RootSpanID != "" {
		response["root_span_id"] = trace.RootSpanID
	}

	c.JSON(http.StatusOK, response)
}

// ─── Document fetch ───────────────────────────────────────────────────────────

// handleGetDocument returns the raw content of a document from the data dir.
// The document is stored under ~/.zen-sci/documents/<id>.json by the portal's
// Rust backend; this endpoint lets the agent loop fetch it without the portal
// being in the request path.
// GET /v1/gateway/documents/:id
func (s *Server) handleGetDocument(c *gin.Context) {
	docID := c.Param("id")
	if docID == "" {
		s.errorResponse(c, http.StatusBadRequest, "invalid_request", "document id is required")
		return
	}

	home, err := os.UserHomeDir()
	if err != nil {
		s.errorResponse(c, http.StatusInternalServerError, "server_error", "cannot determine home directory")
		return
	}

	docPath := filepath.Join(home, ".zen-sci", "documents", docID+".json")
	data, err := os.ReadFile(docPath)
	if err != nil {
		if os.IsNotExist(err) {
			s.errorResponse(c, http.StatusNotFound, "not_found", fmt.Sprintf("document not found: %s", docID))
		} else {
			s.errorResponse(c, http.StatusInternalServerError, "server_error", fmt.Sprintf("failed to read document: %v", err))
		}
		return
	}

	// Parse to validate and re-emit as JSON
	var doc map[string]interface{}
	if err := json.Unmarshal(data, &doc); err != nil {
		s.errorResponse(c, http.StatusInternalServerError, "server_error", "document file is corrupt")
		return
	}

	c.JSON(http.StatusOK, doc)
}

// ─── Provider key management ──────────────────────────────────────────────────

// providerKeysPath returns the path to the provider keys file.
func providerKeysPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".zen-sci", "provider_keys.json"), nil
}

// RestorePersistedProviderKeys reads ~/.zen-sci/provider_keys.json and
// hot-registers any stored provider keys. Called once at startup so cloud
// providers survive a gateway restart without requiring the CLI to re-push keys.
func (s *Server) RestorePersistedProviderKeys() {
	path, err := providerKeysPath()
	if err != nil {
		slog.Warn("RestorePersistedProviderKeys: cannot determine keys path", "error", err)
		return
	}
	data, err := os.ReadFile(path)
	if err != nil {
		// File not present is normal on first boot.
		return
	}
	keys := make(map[string]string)
	if err := json.Unmarshal(data, &keys); err != nil {
		slog.Warn("RestorePersistedProviderKeys: failed to parse keys file", "path", path, "error", err)
		return
	}
	restored := 0
	for provider, key := range keys {
		if key == "" {
			continue
		}
		s.hotRegisterProvider(provider, key)
		slog.Info("restored persisted provider key", "provider", provider)
		restored++
	}
	slog.Info("persisted provider keys restored", "count", restored)
}

// handleSetProviderKey stores or clears a provider API key.
// POST /v1/settings/providers
func (s *Server) handleSetProviderKey(c *gin.Context) {
	var req struct {
		Provider string `json:"provider" binding:"required"`
		Key      string `json:"key"` // empty string = remove key
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		s.errorResponse(c, http.StatusBadRequest, "invalid_request", fmt.Sprintf("invalid request: %v", err))
		return
	}

	path, err := providerKeysPath()
	if err != nil {
		s.errorResponse(c, http.StatusInternalServerError, "server_error", "cannot determine data directory")
		return
	}

	// Read existing keys (if any)
	keys := make(map[string]string)
	if data, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(data, &keys) // ignore parse errors — just start fresh
	}

	if req.Key == "" {
		delete(keys, req.Provider)
	} else {
		keys[req.Provider] = req.Key

		// Hot-register the provider immediately so it is available without restart.
		if s.pluginManager != nil {
			s.hotRegisterProvider(req.Provider, req.Key)
		}

		// If the semantic router exists but hasn't been initialized yet,
		// attempt lazy initialization now that a new provider is available.
		// sync.Once ensures only one goroutine is spawned even on rapid concurrent POSTs.
		if s.semanticRouter != nil && !s.semanticRouter.IsInitialized() {
			s.semanticRouterInitOnce.Do(func() {
				go func() {
					ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
					defer cancel()
					if ok, err := s.semanticRouter.TryInitialize(ctx); err != nil {
						slog.Warn("semantic router auto-init failed", "error", err)
					} else if ok {
						slog.Info("semantic router auto-initialized after provider registration")
					}
				}()
			})
		}
	}

	// Persist
	data, _ := json.MarshalIndent(keys, "", "  ")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		s.errorResponse(c, http.StatusInternalServerError, "server_error", "cannot create data directory")
		return
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		s.errorResponse(c, http.StatusInternalServerError, "server_error", fmt.Sprintf("failed to write keys: %v", err))
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true, "provider": req.Provider, "configured": req.Key != ""})
}

// hotRegisterProvider creates and registers a cloud provider with the given key.
// If the provider is already registered, its BaseProvider.APIKey is updated in place.
// This allows the gateway to activate cloud providers at runtime without a restart.
func (s *Server) hotRegisterProvider(name, apiKey string) {
	// Map provider name → factory function (same mapping as RegisterProviders in services).
	type factory func(string) provider.ModelProvider
	factories := map[string]factory{
		"anthropic":    func(k string) provider.ModelProvider { return providers.NewAnthropicProvider(k) },
		"openai":       func(k string) provider.ModelProvider { return providers.NewOpenAIProvider(k) },
		"google":       func(k string) provider.ModelProvider { return providers.NewGoogleProvider(k) },
		"groq":         func(k string) provider.ModelProvider { return providers.NewGroqProvider(k) },
		"mistral":      func(k string) provider.ModelProvider { return providers.NewMistralProvider(k) },
		"deepseek-api": func(k string) provider.ModelProvider { return providers.NewDeepSeekProvider(k) },
		"kimi":         func(k string) provider.ModelProvider { return providers.NewKimiProvider(k) },
	}

	// If already registered, update the key directly via BaseProvider.
	if existing, ok := s.pluginManager.GetProviders()[name]; ok {
		type keyUpdater interface {
			SetAPIKey(string)
		}
		if ku, ok := existing.(keyUpdater); ok {
			ku.SetAPIKey(apiKey)
		}
		// Whether or not SetAPIKey exists, the key is now in the persistent store.
		// BaseProvider.ResolveAPIKey will pick it up from env if needed.
		return
	}

	// Provider not yet registered — create it and register it now.
	f, known := factories[name]
	if !known {
		return // unknown provider name; nothing to do
	}
	p := f(apiKey)
	s.pluginManager.RegisterProvider(name, p)
}

// handleGetProviderSettings returns which providers have keys configured (no key values).
// GET /v1/settings/providers
func (s *Server) handleGetProviderSettings(c *gin.Context) {
	path, err := providerKeysPath()
	if err != nil {
		s.errorResponse(c, http.StatusInternalServerError, "server_error", "cannot determine data directory")
		return
	}

	keys := make(map[string]string)
	if data, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(data, &keys)
	}

	configured := make(map[string]bool)
	for k, v := range keys {
		configured[k] = v != ""
	}

	c.JSON(http.StatusOK, gin.H{"providers": configured})
}

// ─── Agent listing (Gap 4: filtering support) ──────────────────────────────

// handleGatewayListAgents returns registered agents with optional filters.
// GET /v1/gateway/agents?status=active|inactive|all&model=X&limit=N&offset=N
func (s *Server) handleGatewayListAgents(c *gin.Context) {
	statusFilter := c.DefaultQuery("status", "all")
	modelFilter := c.Query("model")

	limitStr := c.DefaultQuery("limit", "50")
	offsetStr := c.DefaultQuery("offset", "0")

	limit, errLimit := strconv.Atoi(limitStr)
	if errLimit != nil || limit < 1 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}

	offset, errOffset := strconv.Atoi(offsetStr)
	if errOffset != nil || offset < 0 {
		offset = 0
	}

	s.agentMu.RLock()
	defer s.agentMu.RUnlock()

	// Build filtered list
	allAgents := make([]gin.H, 0, len(s.agents))
	for id, runtime := range s.agents {
		// Determine agent status
		agentStatus := "active"
		if runtime.Config == nil {
			agentStatus = "inactive"
		}

		// Apply status filter
		if statusFilter != "all" && agentStatus != statusFilter {
			continue
		}

		// Apply model filter (check if disposition or config contains model info)
		if modelFilter != "" {
			// Model filtering checks Config fields if available
			if runtime.Config != nil {
				// No direct model field on AgentConfig; skip agents that don't match
				// For now, model filter is a pass-through since model is set at chat time
			}
		}

		entry := gin.H{
			"agent_id": id,
			"status":   agentStatus,
		}
		if runtime.Config != nil {
			entry["config"] = runtime.Config
		}
		if runtime.Disposition != nil {
			entry["disposition"] = gin.H{
				"pacing":     runtime.Disposition.Pacing,
				"depth":      runtime.Disposition.Depth,
				"tone":       runtime.Disposition.Tone,
				"initiative": runtime.Disposition.Initiative,
			}
		}
		if len(runtime.Channels) > 0 {
			entry["channels"] = runtime.Channels
		}
		allAgents = append(allAgents, entry)
	}

	total := len(allAgents)

	// Apply pagination
	if offset > total {
		offset = total
	}
	end := offset + limit
	if end > total {
		end = total
	}
	paged := allAgents[offset:end]

	c.JSON(http.StatusOK, gin.H{
		"agents": paged,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

// ─── Agent-channel binding (Gap 5) ──────────────────────────────────────────

// handleGatewayBindAgentChannels stores a list of channels on an agent's metadata.
// POST /v1/gateway/agents/:id/channels
func (s *Server) handleGatewayBindAgentChannels(c *gin.Context) {
	agentID := c.Param("id")

	var req struct {
		Channels []string `json:"channels" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		s.errorResponse(c, http.StatusBadRequest, "invalid_request", fmt.Sprintf("Invalid request: %v", err))
		return
	}

	s.agentMu.Lock()
	runtime, exists := s.agents[agentID]
	if !exists {
		s.agentMu.Unlock()
		s.errorResponse(c, http.StatusNotFound, "not_found", fmt.Sprintf("Agent not found: %s", agentID))
		return
	}

	// Append new channels, deduplicating
	existing := make(map[string]bool, len(runtime.Channels))
	for _, ch := range runtime.Channels {
		existing[ch] = true
	}
	for _, ch := range req.Channels {
		if !existing[ch] {
			runtime.Channels = append(runtime.Channels, ch)
			existing[ch] = true
		}
	}
	result := make([]string, len(runtime.Channels))
	copy(result, runtime.Channels)
	s.agentMu.Unlock()

	c.JSON(http.StatusOK, gin.H{
		"agent_id": agentID,
		"channels": result,
		"count":    len(result),
	})
}

// Agent channel handlers (handleGatewayListAgentChannels, handleGatewayUnbindAgentChannel)
// are defined in handle_gateway_channels.go (Gap 5 extended version).

// ─── Helper functions ─────────────────────────────────────────────────────────

func extractNamespace(toolName string) string {
	// Extract namespace from tool name (e.g., "composio.create_task" → "composio")
	for i, char := range toolName {
		if char == '.' || char == ':' {
			return toolName[:i]
		}
	}
	return "builtin"
}

func convertToOrchestrationPlan(gatewayPlan *gateway.ExecutionPlan) *orchestrationpkg.Plan {
	nodes := make([]*orchestrationpkg.PlanNode, 0, len(gatewayPlan.DAG))

	for _, invocation := range gatewayPlan.DAG {
		node := &orchestrationpkg.PlanNode{
			ID:           invocation.ID,
			ToolName:     invocation.ToolName,
			Parameters:   invocation.Input,
			Dependencies: invocation.DependsOn,
			State:        orchestrationpkg.NodeStatePending,
		}
		nodes = append(nodes, node)
	}

	return &orchestrationpkg.Plan{
		ID:        gatewayPlan.ID,
		TaskID:    gatewayPlan.Name, // Store gateway plan name in TaskID field
		Nodes:     nodes,
		CreatedAt: time.Now(),
		Version:   1,
		Metadata:  map[string]interface{}{"gateway_plan_name": gatewayPlan.Name},
	}
}
