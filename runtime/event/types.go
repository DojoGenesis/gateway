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

// Channel event type constants (ADR-015 addendum, Era 3 Phase 1 Track A).
//
// Subject conventions for JetStream:
//
//	dojo.channel.message.{platform} — inbound normalized ChannelMessage
//	dojo.channel.reply.{platform}   — outbound reply from workflow execution
//
// Each platform has a JetStream durable consumer named "channel-{platform}"
// with MaxAge retention of 30 days.
const (
	// EventChannelMessage is the event type for inbound channel messages.
	// Subject: dojo.channel.message.{platform}
	EventChannelMessage = "dojo.channel.message"

	// EventChannelReply is the event type for outbound channel replies.
	// Subject: dojo.channel.reply.{platform}
	EventChannelReply = "dojo.channel.reply"
)

// ChannelMessageSubject returns the NATS subject for an inbound channel
// message on the given platform: "dojo.channel.message.{platform}".
func ChannelMessageSubject(platform string) string {
	return EventChannelMessage + "." + platform
}

// ChannelReplySubject returns the NATS subject for an outbound reply on
// the given platform: "dojo.channel.reply.{platform}".
func ChannelReplySubject(platform string) string {
	return EventChannelReply + "." + platform
}

// ChannelSubjectWildcard returns a subject pattern matching all channel events.
const ChannelSubjectWildcard = "dojo.channel.>"
