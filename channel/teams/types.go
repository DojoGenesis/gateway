// Package teams implements the Microsoft Teams Bot Framework adapter for the
// Dojo Channel Bridge. It normalizes Bot Framework Activity payloads into
// ChannelMessage envelopes and delivers outbound messages via the Bot
// Connector REST API.
//
// No external Go SDK is used — the adapter calls the REST API directly
// (ADR-018 decision, Phase 0 dependency isolation).
package teams

// Activity represents a Bot Framework v3 Activity object.
// See https://docs.microsoft.com/en-us/azure/bot-service/rest-api/bot-framework-rest-connector-api-reference
type Activity struct {
	Type         string              `json:"type"`
	ID           string              `json:"id"`
	Timestamp    string              `json:"timestamp,omitempty"`
	Text         string              `json:"text,omitempty"`
	From         ChannelAccount      `json:"from"`
	Conversation ConversationAccount `json:"conversation"`
	Recipient    ChannelAccount      `json:"recipient"`
	ReplyToID    string              `json:"replyToId,omitempty"`
	ServiceURL   string              `json:"serviceUrl"`
	Attachments  []Attachment        `json:"attachments,omitempty"`
	ChannelID    string              `json:"channelId,omitempty"`
}

// ChannelAccount identifies a user or bot in the Bot Framework.
type ChannelAccount struct {
	ID   string `json:"id"`
	Name string `json:"name,omitempty"`
}

// ConversationAccount identifies a conversation.
type ConversationAccount struct {
	ID   string `json:"id"`
	Name string `json:"name,omitempty"`
}

// Attachment represents a rich card or file attachment in a Bot Framework Activity.
type Attachment struct {
	ContentType string      `json:"contentType"`
	ContentURL  string      `json:"contentUrl,omitempty"`
	Name        string      `json:"name,omitempty"`
	Content     interface{} `json:"content,omitempty"`
}
