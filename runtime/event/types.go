package event

// Event type constants for the Dojo platform.
const (
	EventToolRequested   = "dojo.tool.requested"
	EventToolCompleted   = "dojo.tool.completed"
	EventToolFailed      = "dojo.tool.failed"
	EventAgentSpawned    = "dojo.agent.spawned"
	EventAgentStopped    = "dojo.agent.stopped"
	EventAgentMessage    = "dojo.agent.message"
	EventWorkflowStarted = "dojo.workflow.started"
	EventWorkflowStep    = "dojo.workflow.step"
	EventWorkflowDone    = "dojo.workflow.done"
	EventSkillInvoked    = "dojo.skill.invoked"
	EventSkillCompleted  = "dojo.skill.completed"
	EventMemoryStored    = "dojo.memory.stored"
)
