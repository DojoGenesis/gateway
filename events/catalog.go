package events

// ─── SSE Event Catalog ──────────────────────────────────────────────────────
//
// This file is the authoritative catalog of all SSE event types emitted by the
// Agentic Gateway. Each event type has a Go constant, a documented JSON payload
// schema, and a constructor function (defined in events.go).
//
// Wire Format (RFC 8895):
//
//	event: <EventType>\n
//	data: <JSON payload>\n
//	\n
//
// All payloads share a common envelope:
//
//	{
//	  "type": "<EventType>",
//	  "data": { ... event-specific fields ... },
//	  "timestamp": "<RFC3339>"
//	}

// ─── Event Categories ───────────────────────────────────────────────────────

// AllEventTypes returns every registered event type for discovery and validation.
func AllEventTypes() []EventType {
	return []EventType{
		// Core agent events (v0.0.17)
		IntentClassified,
		ProviderSelected,
		ToolInvoked,
		ToolCompleted,
		Thinking,
		ResponseChunk,
		MemoryRetrieved,
		Complete,
		Error,
		TraceSpanStart,
		TraceSpanEnd,
		// Artifact engine events (v0.0.18)
		ArtifactCreated,
		ArtifactUpdated,
		ProjectSwitched,
		DiagramRendered,
		// Orchestration engine events (v0.0.30)
		OrchestrationPlanCreated,
		OrchestrationNodeStart,
		OrchestrationNodeEnd,
		OrchestrationReplanning,
		OrchestrationComplete,
		OrchestrationFailed,
	}
}

// IsValidEventType returns true if the given type is a known event type.
func IsValidEventType(t EventType) bool {
	for _, et := range AllEventTypes() {
		if et == t {
			return true
		}
	}
	return false
}

// ─── Payload Schema Documentation ───────────────────────────────────────────
//
// Each event's data field follows the schema documented below.
// Constructor functions in events.go enforce these shapes.
//
// ── Core Agent Events (v0.0.17) ─────────────────────────────────────────────
//
// IntentClassified ("intent_classified")
//   data: {
//     "intent":     string,  // classified intent label
//     "confidence": float64  // 0.0–1.0 confidence score
//   }
//   Constructor: NewIntentClassifiedEvent(intent, confidence)
//
// ProviderSelected ("provider_selected")
//   data: {
//     "provider": string,  // provider name (e.g., "anthropic")
//     "model":    string   // model identifier (e.g., "claude-3-opus")
//   }
//   Constructor: NewProviderSelectedEvent(provider, model)
//
// ToolInvoked ("tool_invoked")
//   data: {
//     "tool":      string,                  // tool name
//     "arguments": map[string]interface{}   // invocation arguments
//   }
//   Constructor: NewToolInvokedEvent(toolName, arguments)
//
// ToolCompleted ("tool_completed")
//   data: {
//     "tool":        string,       // tool name
//     "result":      interface{},  // tool execution result
//     "duration_ms": int64         // execution time in milliseconds
//   }
//   Constructor: NewToolCompletedEvent(toolName, result, durationMs)
//
// Thinking ("thinking")
//   data: {
//     "message": string  // agent reasoning/thinking text
//   }
//   Constructor: NewThinkingEvent(message)
//
// ResponseChunk ("response_chunk")
//   data: {
//     "content": string  // partial response text (~50 words per chunk)
//   }
//   Constructor: NewResponseChunkEvent(content)
//
// MemoryRetrieved ("memory_retrieved")
//   data: {
//     "memories_found": int,            // count of memories found
//     "memories":       []interface{}   // retrieved memory entries
//   }
//   Constructor: NewMemoryRetrievedEvent(memoriesFound, memories)
//
// Complete ("complete")
//   data: {
//     "usage": map[string]interface{}  // token usage stats
//   }
//   Constructor: NewCompleteEvent(usage)
//
// Error ("error")
//   data: {
//     "error":      string,  // human-readable error message
//     "error_code": string   // machine-readable error code
//   }
//   Constructor: NewErrorEvent(errorMessage, errorCode)
//
// TraceSpanStart ("trace_span_start")
//   data: {
//     "trace_id":   string,
//     "span_id":    string,
//     "parent_id":  string,
//     "name":       string,
//     "start_time": time.Time,
//     "inputs":     map[string]interface{}
//   }
//   Constructor: NewTraceSpanStartEvent(traceID, spanID, parentID, name, startTime, inputs)
//
// TraceSpanEnd ("trace_span_end")
//   data: {
//     "trace_id":    string,
//     "span_id":     string,
//     "parent_id":   string,
//     "name":        string,
//     "start_time":  time.Time,
//     "end_time":    *time.Time,
//     "inputs":      map[string]interface{},
//     "outputs":     map[string]interface{},
//     "metadata":    map[string]interface{},
//     "status":      string,
//     "duration_ms": int64
//   }
//   Constructor: NewTraceSpanEndEvent(traceID, spanID, parentID, name, startTime, endTime, inputs, outputs, metadata, status, durationMs)
//
// ── Artifact Engine Events (v0.0.18) ────────────────────────────────────────
//
// ArtifactCreated ("artifact_created")
//   data: {
//     "artifact_id":   string,
//     "artifact_name": string,
//     "artifact_type": string,
//     "project_id":    string
//   }
//   Constructor: NewArtifactCreatedEvent(artifactID, name, type, projectID)
//
// ArtifactUpdated ("artifact_updated")
//   data: {
//     "artifact_id":    string,
//     "artifact_name":  string,
//     "version":        int,
//     "commit_message": string
//   }
//   Constructor: NewArtifactUpdatedEvent(artifactID, name, version, commitMessage)
//
// ProjectSwitched ("project_switched")
//   data: {
//     "project_id":   string,
//     "project_name": string
//   }
//   Constructor: NewProjectSwitchedEvent(projectID, projectName)
//
// DiagramRendered ("diagram_rendered")
//   data: {
//     "diagram_id":   string,
//     "diagram_type": string,
//     "format":       string
//   }
//   Constructor: NewDiagramRenderedEvent(diagramID, diagramType, format)
//
// ── Orchestration Engine Events (v0.0.30) ───────────────────────────────────
//
// OrchestrationPlanCreated ("orchestration_plan_created")
//   data: {
//     "plan_id":        string,
//     "task_id":        string,
//     "node_count":     int,
//     "estimated_cost": float64,
//     "plan":           interface{}  // serialized plan structure
//   }
//   Constructor: NewOrchestrationPlanCreatedEvent(planID, taskID, nodeCount, estimatedCost, plan)
//
// OrchestrationNodeStart ("orchestration_node_start")
//   data: {
//     "node_id":    string,
//     "plan_id":    string,
//     "tool_name":  string,
//     "parameters": map[string]interface{}
//   }
//   Constructor: NewOrchestrationNodeStartEvent(nodeID, planID, toolName, parameters)
//
// OrchestrationNodeEnd ("orchestration_node_end")
//   data: {
//     "node_id":     string,
//     "plan_id":     string,
//     "tool_name":   string,
//     "state":       string,       // "success" | "failed" | "skipped"
//     "result":      interface{},
//     "error":       string,
//     "duration_ms": int64
//   }
//   Constructor: NewOrchestrationNodeEndEvent(nodeID, planID, toolName, state, result, errorMsg, durationMs)
//
// OrchestrationReplanning ("orchestration_replanning")
//   data: {
//     "plan_id":      string,
//     "task_id":      string,
//     "reason":       string,
//     "failed_nodes": []string
//   }
//   Constructor: NewOrchestrationReplanningEvent(planID, taskID, reason, failedNodes)
//
// OrchestrationComplete ("orchestration_complete")
//   data: {
//     "plan_id":       string,
//     "task_id":       string,
//     "total_nodes":   int,
//     "success_nodes": int,
//     "failed_nodes":  int,
//     "duration_ms":   int64
//   }
//   Constructor: NewOrchestrationCompleteEvent(planID, taskID, totalNodes, successNodes, failedNodes, durationMs)
//
// OrchestrationFailed ("orchestration_failed")
//   data: {
//     "plan_id": string,
//     "task_id": string,
//     "reason":  string
//   }
//   Constructor: NewOrchestrationFailedEvent(planID, taskID, reason)
