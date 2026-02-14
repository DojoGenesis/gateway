package events

import (
	"encoding/json"
	"time"
)

// EventType identifies the kind of server-sent event emitted by the gateway.
// See catalog.go for the full list of types and their documented payload schemas.
type EventType string

const (
	IntentClassified EventType = "intent_classified"
	ProviderSelected EventType = "provider_selected"
	ToolInvoked      EventType = "tool_invoked"
	ToolCompleted    EventType = "tool_completed"
	Thinking         EventType = "thinking"
	ResponseChunk    EventType = "response_chunk"
	MemoryRetrieved  EventType = "memory_retrieved"
	Complete         EventType = "complete"
	Error            EventType = "error"
	TraceSpanStart   EventType = "trace_span_start"
	TraceSpanEnd     EventType = "trace_span_end"
	// Artifact Engine events
	ArtifactCreated EventType = "artifact_created"
	ArtifactUpdated EventType = "artifact_updated"
	ProjectSwitched EventType = "project_switched"
	DiagramRendered EventType = "diagram_rendered"
	// Orchestration Engine events
	OrchestrationPlanCreated EventType = "orchestration_plan_created"
	OrchestrationNodeStart   EventType = "orchestration_node_start"
	OrchestrationNodeEnd     EventType = "orchestration_node_end"
	OrchestrationReplanning  EventType = "orchestration_replanning"
	OrchestrationComplete    EventType = "orchestration_complete"
	OrchestrationFailed      EventType = "orchestration_failed"
)

// StreamEvent is the envelope for all SSE events sent by the gateway.
// It contains the event type, a data payload (schema varies by type —
// see catalog.go), and a timestamp.
type StreamEvent struct {
	Type      EventType              `json:"type"`
	Data      map[string]interface{} `json:"data"`
	Timestamp time.Time              `json:"timestamp"`
}

func NewIntentClassifiedEvent(intent string, confidence float64) StreamEvent {
	return StreamEvent{
		Type: IntentClassified,
		Data: map[string]interface{}{
			"intent":     intent,
			"confidence": confidence,
		},
		Timestamp: time.Now(),
	}
}

func NewProviderSelectedEvent(provider, model string) StreamEvent {
	return StreamEvent{
		Type: ProviderSelected,
		Data: map[string]interface{}{
			"provider": provider,
			"model":    model,
		},
		Timestamp: time.Now(),
	}
}

func NewToolInvokedEvent(toolName string, arguments map[string]interface{}) StreamEvent {
	return StreamEvent{
		Type: ToolInvoked,
		Data: map[string]interface{}{
			"tool":      toolName,
			"arguments": arguments,
		},
		Timestamp: time.Now(),
	}
}

func NewToolCompletedEvent(toolName string, result interface{}, durationMs int64) StreamEvent {
	return StreamEvent{
		Type: ToolCompleted,
		Data: map[string]interface{}{
			"tool":        toolName,
			"result":      result,
			"duration_ms": durationMs,
		},
		Timestamp: time.Now(),
	}
}

func NewThinkingEvent(message string) StreamEvent {
	return StreamEvent{
		Type: Thinking,
		Data: map[string]interface{}{
			"message": message,
		},
		Timestamp: time.Now(),
	}
}

func NewResponseChunkEvent(content string) StreamEvent {
	return StreamEvent{
		Type: ResponseChunk,
		Data: map[string]interface{}{
			"content": content,
		},
		Timestamp: time.Now(),
	}
}

func NewMemoryRetrievedEvent(memoriesFound int, memories []interface{}) StreamEvent {
	return StreamEvent{
		Type: MemoryRetrieved,
		Data: map[string]interface{}{
			"memories_found": memoriesFound,
			"memories":       memories,
		},
		Timestamp: time.Now(),
	}
}

func NewCompleteEvent(usage map[string]interface{}) StreamEvent {
	return StreamEvent{
		Type: Complete,
		Data: map[string]interface{}{
			"usage": usage,
		},
		Timestamp: time.Now(),
	}
}

func NewErrorEvent(errorMessage string, errorCode string) StreamEvent {
	return StreamEvent{
		Type: Error,
		Data: map[string]interface{}{
			"error":      errorMessage,
			"error_code": errorCode,
		},
		Timestamp: time.Now(),
	}
}

func NewTraceSpanStartEvent(traceID, spanID, parentID, name string, startTime time.Time, inputs map[string]interface{}) StreamEvent {
	return StreamEvent{
		Type: TraceSpanStart,
		Data: map[string]interface{}{
			"trace_id":   traceID,
			"span_id":    spanID,
			"parent_id":  parentID,
			"name":       name,
			"start_time": startTime,
			"inputs":     inputs,
		},
		Timestamp: time.Now(),
	}
}

func NewTraceSpanEndEvent(traceID, spanID, parentID, name string, startTime time.Time, endTime *time.Time, inputs, outputs, metadata map[string]interface{}, status string, durationMs int64) StreamEvent {
	return StreamEvent{
		Type: TraceSpanEnd,
		Data: map[string]interface{}{
			"trace_id":    traceID,
			"span_id":     spanID,
			"parent_id":   parentID,
			"name":        name,
			"start_time":  startTime,
			"end_time":    endTime,
			"inputs":      inputs,
			"outputs":     outputs,
			"metadata":    metadata,
			"status":      status,
			"duration_ms": durationMs,
		},
		Timestamp: time.Now(),
	}
}

func (e StreamEvent) ToJSON() ([]byte, error) {
	return json.Marshal(e)
}

func (e StreamEvent) String() string {
	data, err := e.ToJSON()
	if err != nil {
		return ""
	}
	return string(data)
}

func FromJSON(data []byte) (*StreamEvent, error) {
	var event StreamEvent
	err := json.Unmarshal(data, &event)
	if err != nil {
		return nil, err
	}
	return &event, nil
}

// Artifact Engine event constructors
func NewArtifactCreatedEvent(artifactID, artifactName, artifactType, projectID string) StreamEvent {
	return StreamEvent{
		Type: ArtifactCreated,
		Data: map[string]interface{}{
			"artifact_id":   artifactID,
			"artifact_name": artifactName,
			"artifact_type": artifactType,
			"project_id":    projectID,
		},
		Timestamp: time.Now(),
	}
}

func NewArtifactUpdatedEvent(artifactID, artifactName string, version int, commitMessage string) StreamEvent {
	return StreamEvent{
		Type: ArtifactUpdated,
		Data: map[string]interface{}{
			"artifact_id":    artifactID,
			"artifact_name":  artifactName,
			"version":        version,
			"commit_message": commitMessage,
		},
		Timestamp: time.Now(),
	}
}

func NewProjectSwitchedEvent(projectID, projectName string) StreamEvent {
	return StreamEvent{
		Type: ProjectSwitched,
		Data: map[string]interface{}{
			"project_id":   projectID,
			"project_name": projectName,
		},
		Timestamp: time.Now(),
	}
}

func NewDiagramRenderedEvent(diagramID, diagramType, format string) StreamEvent {
	return StreamEvent{
		Type: DiagramRendered,
		Data: map[string]interface{}{
			"diagram_id":   diagramID,
			"diagram_type": diagramType,
			"format":       format,
		},
		Timestamp: time.Now(),
	}
}

func NewOrchestrationPlanCreatedEvent(planID, taskID string, nodeCount int, estimatedCost float64, plan interface{}) StreamEvent {
	return StreamEvent{
		Type: OrchestrationPlanCreated,
		Data: map[string]interface{}{
			"plan_id":        planID,
			"task_id":        taskID,
			"node_count":     nodeCount,
			"estimated_cost": estimatedCost,
			"plan":           plan,
		},
		Timestamp: time.Now(),
	}
}

func NewOrchestrationNodeStartEvent(nodeID, planID, toolName string, parameters map[string]interface{}) StreamEvent {
	return StreamEvent{
		Type: OrchestrationNodeStart,
		Data: map[string]interface{}{
			"node_id":    nodeID,
			"plan_id":    planID,
			"tool_name":  toolName,
			"parameters": parameters,
		},
		Timestamp: time.Now(),
	}
}

func NewOrchestrationNodeEndEvent(nodeID, planID, toolName, state string, result interface{}, errorMsg string, durationMs int64) StreamEvent {
	return StreamEvent{
		Type: OrchestrationNodeEnd,
		Data: map[string]interface{}{
			"node_id":     nodeID,
			"plan_id":     planID,
			"tool_name":   toolName,
			"state":       state,
			"result":      result,
			"error":       errorMsg,
			"duration_ms": durationMs,
		},
		Timestamp: time.Now(),
	}
}

func NewOrchestrationReplanningEvent(planID, taskID, reason string, failedNodes []string) StreamEvent {
	return StreamEvent{
		Type: OrchestrationReplanning,
		Data: map[string]interface{}{
			"plan_id":      planID,
			"task_id":      taskID,
			"reason":       reason,
			"failed_nodes": failedNodes,
		},
		Timestamp: time.Now(),
	}
}

func NewOrchestrationCompleteEvent(planID, taskID string, totalNodes, successNodes, failedNodes int, durationMs int64) StreamEvent {
	return StreamEvent{
		Type: OrchestrationComplete,
		Data: map[string]interface{}{
			"plan_id":       planID,
			"task_id":       taskID,
			"total_nodes":   totalNodes,
			"success_nodes": successNodes,
			"failed_nodes":  failedNodes,
			"duration_ms":   durationMs,
		},
		Timestamp: time.Now(),
	}
}

func NewOrchestrationFailedEvent(planID, taskID, reason string) StreamEvent {
	return StreamEvent{
		Type: OrchestrationFailed,
		Data: map[string]interface{}{
			"plan_id": planID,
			"task_id": taskID,
			"reason":  reason,
		},
		Timestamp: time.Now(),
	}
}
