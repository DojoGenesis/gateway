package streaming

import (
	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/events"
)

type EventType = events.EventType

const (
	// v0.0.17: Core events
	IntentClassified = events.IntentClassified
	ProviderSelected = events.ProviderSelected
	ToolInvoked      = events.ToolInvoked
	ToolCompleted    = events.ToolCompleted
	Thinking         = events.Thinking
	ResponseChunk    = events.ResponseChunk
	MemoryRetrieved  = events.MemoryRetrieved
	Complete         = events.Complete
	Error            = events.Error
	TraceSpanStart   = events.TraceSpanStart
	TraceSpanEnd     = events.TraceSpanEnd
	// v1.2: Agent Chat Streaming events
	PatchIntentEvent = events.PatchIntentEvent
	// v0.0.18: Artifact Engine events
	ArtifactCreated = events.ArtifactCreated
	ArtifactUpdated = events.ArtifactUpdated
	ProjectSwitched = events.ProjectSwitched
	DiagramRendered = events.DiagramRendered
	// v0.0.30: Orchestration Engine events
	OrchestrationPlanCreated = events.OrchestrationPlanCreated
	OrchestrationNodeStart   = events.OrchestrationNodeStart
	OrchestrationNodeEnd     = events.OrchestrationNodeEnd
	OrchestrationReplanning  = events.OrchestrationReplanning
	OrchestrationComplete    = events.OrchestrationComplete
	OrchestrationFailed      = events.OrchestrationFailed
)

type StreamEvent = events.StreamEvent

var (
	// v0.0.17: Core event constructors
	NewIntentClassifiedEvent = events.NewIntentClassifiedEvent
	NewProviderSelectedEvent = events.NewProviderSelectedEvent
	NewToolInvokedEvent      = events.NewToolInvokedEvent
	NewToolCompletedEvent    = events.NewToolCompletedEvent
	NewThinkingEvent         = events.NewThinkingEvent
	NewResponseChunkEvent    = events.NewResponseChunkEvent
	NewMemoryRetrievedEvent  = events.NewMemoryRetrievedEvent
	NewCompleteEvent         = events.NewCompleteEvent
	NewErrorEvent            = events.NewErrorEvent
	NewTraceSpanStartEvent   = events.NewTraceSpanStartEvent
	NewTraceSpanEndEvent     = events.NewTraceSpanEndEvent
	// v1.2: Agent Chat Streaming event constructors
	NewPatchIntentEvent = events.NewPatchIntentEvent
	// v0.0.18: Artifact Engine event constructors
	NewArtifactCreatedEvent = events.NewArtifactCreatedEvent
	NewArtifactUpdatedEvent = events.NewArtifactUpdatedEvent
	NewProjectSwitchedEvent = events.NewProjectSwitchedEvent
	NewDiagramRenderedEvent = events.NewDiagramRenderedEvent
	// v0.0.30: Orchestration Engine event constructors
	NewOrchestrationPlanCreatedEvent = events.NewOrchestrationPlanCreatedEvent
	NewOrchestrationNodeStartEvent   = events.NewOrchestrationNodeStartEvent
	NewOrchestrationNodeEndEvent     = events.NewOrchestrationNodeEndEvent
	NewOrchestrationReplanningEvent  = events.NewOrchestrationReplanningEvent
	NewOrchestrationCompleteEvent    = events.NewOrchestrationCompleteEvent
	NewOrchestrationFailedEvent      = events.NewOrchestrationFailedEvent
	FromJSON                         = events.FromJSON
)
