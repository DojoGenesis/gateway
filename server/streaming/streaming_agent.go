package streaming

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/server/agent"
)

const (
	DefaultChannelBufferSize = 100
	ChunkSize                = 50
	ChunkDelay               = 50 * time.Millisecond
)

type StreamingAgent struct {
	agent *agent.PrimaryAgent
}

func NewStreamingAgent(primaryAgent *agent.PrimaryAgent) *StreamingAgent {
	return &StreamingAgent{
		agent: primaryAgent,
	}
}

func (sa *StreamingAgent) HandleQueryStreaming(ctx context.Context, req agent.QueryRequest) (<-chan StreamEvent, error) {
	if sa.agent == nil {
		return nil, fmt.Errorf("primary agent is nil")
	}

	eventChan := make(chan StreamEvent, DefaultChannelBufferSize)

	go func() {
		defer close(eventChan)
		defer func() {
			if r := recover(); r != nil {
				eventChan <- NewErrorEvent(fmt.Sprintf("panic: %v", r), "PANIC")
			}
		}()

		if err := sa.streamQuery(ctx, req, eventChan); err != nil {
			select {
			case eventChan <- NewErrorEvent(err.Error(), "QUERY_ERROR"):
			case <-ctx.Done():
			}
		}
	}()

	return eventChan, nil
}

func (sa *StreamingAgent) streamQuery(ctx context.Context, req agent.QueryRequest, eventChan chan<- StreamEvent) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	startTime := time.Now()

	var response *agent.Response
	var err error

	// Retry up to 2 times for transient errors (deadline exceeded, temporary network issues)
	for attempt := 0; attempt <= 2; attempt++ {
		if attempt > 0 {
			slog.Warn("retrying query", "component", "streaming_agent", "attempt", attempt, "max_attempts", 2, "error", err)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(time.Duration(attempt) * 2 * time.Second):
			}
		}
		response, err = sa.agent.HandleQueryWithTools(ctx, req)
		if err == nil {
			break
		}
		if !isRetryableError(err) {
			return fmt.Errorf("HandleQueryWithTools failed: %w", err)
		}
	}
	if err != nil {
		return fmt.Errorf("HandleQueryWithTools failed after retries: %w", err)
	}

	elapsedMs := time.Since(startTime).Milliseconds()

	if response.Content != "" {
		chunks := splitIntoChunks(response.Content, ChunkSize)
		for _, chunk := range chunks {
			select {
			case eventChan <- NewResponseChunkEvent(chunk):
				time.Sleep(ChunkDelay)
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}

	usageMap := map[string]interface{}{
		"input_tokens":  response.Usage.InputTokens,
		"output_tokens": response.Usage.OutputTokens,
		"total_tokens":  response.Usage.TotalTokens,
		"model":         response.Model,
		"provider":      response.Provider,
		"duration_ms":   elapsedMs,
	}

	select {
	case eventChan <- NewCompleteEvent(usageMap):
	case <-ctx.Done():
		return ctx.Err()
	}

	return nil
}

type StreamingAgentWithEvents struct {
	agent *agent.PrimaryAgent
}

func NewStreamingAgentWithEvents(primaryAgent *agent.PrimaryAgent) *StreamingAgentWithEvents {
	return &StreamingAgentWithEvents{
		agent: primaryAgent,
	}
}

func (sa *StreamingAgentWithEvents) HandleQueryStreamingWithEvents(ctx context.Context, req agent.QueryRequest) (<-chan StreamEvent, error) {
	if sa.agent == nil {
		return nil, fmt.Errorf("primary agent is nil")
	}

	eventChan := make(chan StreamEvent, DefaultChannelBufferSize)

	go func() {
		defer close(eventChan)
		defer func() {
			if r := recover(); r != nil {
				eventChan <- NewErrorEvent(fmt.Sprintf("panic: %v", r), "PANIC")
			}
		}()

		if err := sa.streamQueryWithDetailedEvents(ctx, req, eventChan); err != nil {
			select {
			case eventChan <- NewErrorEvent(err.Error(), "QUERY_ERROR"):
			case <-ctx.Done():
			}
		}
	}()

	return eventChan, nil
}

func (sa *StreamingAgentWithEvents) streamQueryWithDetailedEvents(ctx context.Context, req agent.QueryRequest, eventChan chan<- StreamEvent) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	startTime := time.Now()

	intent, confidence := sa.agent.ClassifyIntent(ctx, req.Query)
	select {
	case eventChan <- NewIntentClassifiedEvent(string(intent), confidence):
	case <-ctx.Done():
		return ctx.Err()
	}

	providerName := req.ProviderName
	if providerName == "" {
		providerName = sa.agent.SelectProvider(req.UserTier, intent)
	}

	modelID := req.ModelID
	if modelID == "" {
		modelID = "default"
	}

	select {
	case eventChan <- NewProviderSelectedEvent(providerName, modelID):
	case <-ctx.Done():
		return ctx.Err()
	}

	if req.UseMemory && req.UserID != "" {
		select {
		case eventChan <- NewMemoryRetrievedEvent(0, []interface{}{}):
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	select {
	case eventChan <- NewThinkingEvent("Processing your request..."):
	case <-ctx.Done():
		return ctx.Err()
	}

	var response *agent.Response
	var err error

	// Retry up to 2 times for transient errors (deadline exceeded, temporary network issues)
	for attempt := 0; attempt <= 2; attempt++ {
		if attempt > 0 {
			slog.Warn("retrying query", "component", "streaming_agent_with_events", "attempt", attempt, "max_attempts", 2, "error", err)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(time.Duration(attempt) * 2 * time.Second):
			}
		}
		response, err = sa.agent.HandleQueryWithTools(ctx, req)
		if err == nil {
			break
		}
		if !isRetryableError(err) {
			return fmt.Errorf("HandleQueryWithTools failed: %w", err)
		}
	}
	if err != nil {
		return fmt.Errorf("HandleQueryWithTools failed after retries: %w", err)
	}

	if len(response.ToolCalls) > 0 {
		for i, tc := range response.ToolCalls {
			select {
			case eventChan <- NewToolInvokedEvent(tc.Name, tc.Arguments):
			case <-ctx.Done():
				return ctx.Err()
			}

			select {
			case eventChan <- NewToolCompletedEvent(tc.Name, "success", 0):
			case <-ctx.Done():
				return ctx.Err()
			}

			// v0.0.18: Emit specific events for artifact and project operations
			// Tool results are indexed to match tool calls (see executeToolCalls in primary_agent.go)
			if i < len(response.ToolResults) {
				result := response.ToolResults[i]
				if result.Error == nil && result.Result != nil {
					if err := sa.emitSpecializedEvent(ctx, tc.Name, result.Result, eventChan); err != nil {
						return err
					}
				}
			}
		}
	}

	if response.Content != "" {
		chunks := splitIntoChunks(response.Content, ChunkSize)
		for _, chunk := range chunks {
			select {
			case eventChan <- NewResponseChunkEvent(chunk):
				time.Sleep(ChunkDelay)
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}

	elapsedMs := time.Since(startTime).Milliseconds()
	usageMap := map[string]interface{}{
		"input_tokens":  response.Usage.InputTokens,
		"output_tokens": response.Usage.OutputTokens,
		"total_tokens":  response.Usage.TotalTokens,
		"model":         response.Model,
		"provider":      response.Provider,
		"duration_ms":   elapsedMs,
	}

	select {
	case eventChan <- NewCompleteEvent(usageMap):
	case <-ctx.Done():
		return ctx.Err()
	}

	return nil
}

// emitSpecializedEvent emits v0.0.18 SSE events for artifact, project, and diagram operations.
//
// This function analyzes tool execution results and emits specialized SSE events to notify
// frontend clients about important state changes:
//   - create_artifact → ArtifactCreated event
//   - update_artifact → ArtifactUpdated event
//   - switch_project → ProjectSwitched event
//   - prepare_diagram (and export_diagram) → DiagramRendered event
//
// Expected tool response structures:
//   - create_artifact: {"success": true, "artifact": {"id", "name", "type", "project_id"}}
//   - update_artifact: {"success": true, "artifact": {"id", "name", "latest_version"}, "commit_message": "..."}
//   - switch_project: {"success": true, "project": {"id", "name"}}
//   - prepare_diagram (and export_diagram): {"success": true, "metadata": {"diagram_id"}, "type", "format"}
//
// If expected fields are missing or success is false, the function silently returns nil
// without emitting an event. This is by design to avoid errors when tools change their
// response structure, but makes debugging difficult.
func (sa *StreamingAgentWithEvents) emitSpecializedEvent(ctx context.Context, toolName string, result map[string]interface{}, eventChan chan<- StreamEvent) error {
	success, ok := result["success"].(bool)
	if !ok || !success {
		slog.Warn("tool completed without success", "component", "sse", "tool", toolName, "success", success)
		return nil
	}

	switch toolName {
	case "create_artifact":
		if artifact, ok := result["artifact"].(map[string]interface{}); ok {
			artifactID := getStringFromMap(artifact, "id")
			artifactName := getStringFromMap(artifact, "name")
			artifactType := getStringFromMap(artifact, "type")
			projectID := getStringFromMap(artifact, "project_id")

			select {
			case eventChan <- NewArtifactCreatedEvent(artifactID, artifactName, artifactType, projectID):
			case <-ctx.Done():
				return ctx.Err()
			}
		} else {
			slog.Warn("missing artifact object in tool result", "component", "sse", "tool", "create_artifact", "event", "ArtifactCreated")
		}

	case "update_artifact":
		if artifact, ok := result["artifact"].(map[string]interface{}); ok {
			artifactID := getStringFromMap(artifact, "id")
			artifactName := getStringFromMap(artifact, "name")
			version := getIntFromMap(artifact, "latest_version")
			commitMessage := getStringFromMap(result, "commit_message")

			select {
			case eventChan <- NewArtifactUpdatedEvent(artifactID, artifactName, version, commitMessage):
			case <-ctx.Done():
				return ctx.Err()
			}
		} else {
			slog.Warn("missing artifact object in tool result", "component", "sse", "tool", "update_artifact", "event", "ArtifactUpdated")
		}

	case "switch_project":
		if project, ok := result["project"].(map[string]interface{}); ok {
			projectID := getStringFromMap(project, "id")
			projectName := getStringFromMap(project, "name")

			select {
			case eventChan <- NewProjectSwitchedEvent(projectID, projectName):
			case <-ctx.Done():
				return ctx.Err()
			}
		} else {
			slog.Warn("missing project object in tool result", "component", "sse", "tool", "switch_project", "event", "ProjectSwitched")
		}

	case "prepare_diagram", "export_diagram":
		if metadata, ok := result["metadata"].(map[string]interface{}); ok {
			diagramID := getStringFromMap(metadata, "diagram_id")
			diagramType := getStringFromMap(result, "type")
			format := getStringFromMap(result, "format")

			select {
			case eventChan <- NewDiagramRenderedEvent(diagramID, diagramType, format):
			case <-ctx.Done():
				return ctx.Err()
			}
		} else {
			slog.Warn("missing metadata object in tool result", "component", "sse", "tool", toolName, "event", "DiagramRendered")
		}
	}

	return nil
}

// Helper functions to safely extract values from map[string]interface{}

// getStringFromMap safely extracts a string value from a map.
// Returns empty string if the key doesn't exist or value is not a string.
func getStringFromMap(m map[string]interface{}, key string) string {
	if val, ok := m[key].(string); ok {
		return val
	}
	return ""
}

// getIntFromMap safely extracts an integer value from a map.
// Returns 0 if the key doesn't exist or value cannot be converted to int.
//
// Note: JSON unmarshaling typically produces float64 for all numbers, so we check
// float64 first (most common case). The int case handles values that were set
// programmatically in Go without going through JSON marshaling (e.g., in tests
// or direct struct initialization).
func getIntFromMap(m map[string]interface{}, key string) int {
	if val, ok := m[key].(float64); ok {
		return int(val)
	}
	if val, ok := m[key].(int); ok {
		return val
	}
	return 0
}

// isRetryableError returns true for transient errors that may succeed on retry.
func isRetryableError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "DeadlineExceeded") ||
		strings.Contains(msg, "context deadline exceeded") ||
		strings.Contains(msg, "connection reset") ||
		strings.Contains(msg, "connection refused") ||
		strings.Contains(msg, "temporary failure") ||
		strings.Contains(msg, "unavailable")
}

func splitIntoChunks(text string, chunkSize int) []string {
	if text == "" {
		return []string{}
	}

	if chunkSize <= 0 {
		chunkSize = ChunkSize
	}

	words := strings.Fields(text)
	if len(words) == 0 {
		return []string{text}
	}

	var chunks []string
	var currentChunk strings.Builder

	for _, word := range words {
		if currentChunk.Len() > 0 && currentChunk.Len()+len(word)+1 > chunkSize {
			chunks = append(chunks, currentChunk.String())
			currentChunk.Reset()
		}

		if currentChunk.Len() > 0 {
			currentChunk.WriteString(" ")
		}
		currentChunk.WriteString(word)
	}

	if currentChunk.Len() > 0 {
		chunks = append(chunks, currentChunk.String())
	}

	return chunks
}
