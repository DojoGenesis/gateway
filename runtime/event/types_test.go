package event_test

import (
	"testing"

	"github.com/DojoGenesis/gateway/runtime/event"
)

// TestChannelMessageSubject verifies that ChannelMessageSubject returns the
// canonical "dojo.channel.message.{platform}" subject for each platform.
func TestChannelMessageSubject(t *testing.T) {
	t.Parallel()

	tests := []struct {
		platform string
		want     string
	}{
		{platform: "slack", want: "dojo.channel.message.slack"},
		{platform: "discord", want: "dojo.channel.message.discord"},
		{platform: "telegram", want: "dojo.channel.message.telegram"},
		{platform: "email", want: "dojo.channel.message.email"},
		{platform: "sms", want: "dojo.channel.message.sms"},
		{platform: "whatsapp", want: "dojo.channel.message.whatsapp"},
		{platform: "teams", want: "dojo.channel.message.teams"},
		{platform: "webchat", want: "dojo.channel.message.webchat"},
		// Edge cases
		{platform: "", want: "dojo.channel.message."},
		{platform: "my-custom-platform", want: "dojo.channel.message.my-custom-platform"},
	}

	for _, tc := range tests {
		t.Run(tc.platform, func(t *testing.T) {
			t.Parallel()
			got := event.ChannelMessageSubject(tc.platform)
			if got != tc.want {
				t.Errorf("ChannelMessageSubject(%q) = %q; want %q", tc.platform, got, tc.want)
			}
		})
	}
}

// TestChannelReplySubject verifies that ChannelReplySubject returns the
// canonical "dojo.channel.reply.{platform}" subject for each platform.
func TestChannelReplySubject(t *testing.T) {
	t.Parallel()

	tests := []struct {
		platform string
		want     string
	}{
		{platform: "slack", want: "dojo.channel.reply.slack"},
		{platform: "discord", want: "dojo.channel.reply.discord"},
		{platform: "telegram", want: "dojo.channel.reply.telegram"},
		{platform: "email", want: "dojo.channel.reply.email"},
		{platform: "sms", want: "dojo.channel.reply.sms"},
		{platform: "whatsapp", want: "dojo.channel.reply.whatsapp"},
		{platform: "teams", want: "dojo.channel.reply.teams"},
		{platform: "webchat", want: "dojo.channel.reply.webchat"},
		// Edge cases
		{platform: "", want: "dojo.channel.reply."},
		{platform: "my-custom-platform", want: "dojo.channel.reply.my-custom-platform"},
	}

	for _, tc := range tests {
		t.Run(tc.platform, func(t *testing.T) {
			t.Parallel()
			got := event.ChannelReplySubject(tc.platform)
			if got != tc.want {
				t.Errorf("ChannelReplySubject(%q) = %q; want %q", tc.platform, got, tc.want)
			}
		})
	}
}

// TestSubjectFunctionsUseConstantPrefixes verifies that the subject-building
// functions are consistent with the exported constant values, so that callers
// using the constants directly get the same prefixes as callers using the
// helper functions.
func TestSubjectFunctionsUseConstantPrefixes(t *testing.T) {
	t.Parallel()

	const platform = "slack"

	msgSubject := event.ChannelMessageSubject(platform)
	wantMsgPrefix := event.EventChannelMessage + "."
	if len(msgSubject) < len(wantMsgPrefix) || msgSubject[:len(wantMsgPrefix)] != wantMsgPrefix {
		t.Errorf("ChannelMessageSubject(%q) = %q; prefix must be %q", platform, msgSubject, wantMsgPrefix)
	}

	replySubject := event.ChannelReplySubject(platform)
	wantReplyPrefix := event.EventChannelReply + "."
	if len(replySubject) < len(wantReplyPrefix) || replySubject[:len(wantReplyPrefix)] != wantReplyPrefix {
		t.Errorf("ChannelReplySubject(%q) = %q; prefix must be %q", platform, replySubject, wantReplyPrefix)
	}
}

// TestChannelSubjectWildcard verifies the wildcard constant value.
func TestChannelSubjectWildcard(t *testing.T) {
	t.Parallel()

	const want = "dojo.channel.>"
	if event.ChannelSubjectWildcard != want {
		t.Errorf("ChannelSubjectWildcard = %q; want %q", event.ChannelSubjectWildcard, want)
	}
}

// TestEventTypeConstants verifies the string values of the core event type
// constants exported by the types package. These are load-bearing in NATS
// subject routing and any change is a breaking API change.
func TestEventTypeConstants(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		got  string
		want string
	}{
		{"EventToolRequested", event.EventToolRequested, "dojo.tool.requested"},
		{"EventToolCompleted", event.EventToolCompleted, "dojo.tool.completed"},
		{"EventToolFailed", event.EventToolFailed, "dojo.tool.failed"},
		{"EventAgentSpawned", event.EventAgentSpawned, "dojo.agent.spawned"},
		{"EventAgentStopped", event.EventAgentStopped, "dojo.agent.stopped"},
		{"EventAgentMessage", event.EventAgentMessage, "dojo.agent.message"},
		{"EventWorkflowStarted", event.EventWorkflowStarted, "dojo.workflow.started"},
		{"EventWorkflowStep", event.EventWorkflowStep, "dojo.workflow.step"},
		{"EventWorkflowDone", event.EventWorkflowDone, "dojo.workflow.done"},
		{"EventSkillInvoked", event.EventSkillInvoked, "dojo.skill.invoked"},
		{"EventSkillCompleted", event.EventSkillCompleted, "dojo.skill.completed"},
		{"EventMemoryStored", event.EventMemoryStored, "dojo.memory.stored"},
		{"EventChannelMessage", event.EventChannelMessage, "dojo.channel.message"},
		{"EventChannelReply", event.EventChannelReply, "dojo.channel.reply"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if tc.got != tc.want {
				t.Errorf("%s = %q; want %q", tc.name, tc.got, tc.want)
			}
		})
	}
}
