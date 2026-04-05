// Package telegram implements the Telegram Bot API adapter for the Dojo
// Channel Bridge. It normalizes Telegram webhook updates into ChannelMessage
// envelopes and delivers outbound messages via the sendMessage API.
//
// This adapter uses net/http + encoding/json directly against the Telegram
// Bot API — no external Telegram library is required (Phase 0 dependency
// isolation).
package telegram

// Update is the top-level payload that Telegram sends to a registered webhook.
// See https://core.telegram.org/bots/api#update
type Update struct {
	UpdateID int      `json:"update_id"`
	Message  *Message `json:"message,omitempty"`
}

// Message represents an individual Telegram message.
// See https://core.telegram.org/bots/api#message
type Message struct {
	MessageID      int         `json:"message_id"`
	From           *User       `json:"from,omitempty"`
	Chat           *Chat       `json:"chat"`
	Date           int64       `json:"date"`
	Text           string      `json:"text,omitempty"`
	ReplyToMessage *Message    `json:"reply_to_message,omitempty"`
	Document       *Document   `json:"document,omitempty"`
	Photo          []PhotoSize `json:"photo,omitempty"`
}

// User represents a Telegram user or bot.
// See https://core.telegram.org/bots/api#user
type User struct {
	ID        int64  `json:"id"`
	IsBot     bool   `json:"is_bot"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name,omitempty"`
	Username  string `json:"username,omitempty"`
}

// Chat represents a Telegram chat (private, group, supergroup, or channel).
// See https://core.telegram.org/bots/api#chat
type Chat struct {
	ID    int64  `json:"id"`
	Type  string `json:"type"`
	Title string `json:"title,omitempty"`
}

// Document represents a general file attachment.
// See https://core.telegram.org/bots/api#document
type Document struct {
	FileID       string    `json:"file_id"`
	FileUniqueID string    `json:"file_unique_id"`
	FileName     string    `json:"file_name,omitempty"`
	MimeType     string    `json:"mime_type,omitempty"`
	FileSize     int64     `json:"file_size,omitempty"`
	Thumbnail    *PhotoSize `json:"thumbnail,omitempty"`
}

// PhotoSize represents one size of a photo or a file/sticker thumbnail.
// See https://core.telegram.org/bots/api#photosize
type PhotoSize struct {
	FileID       string `json:"file_id"`
	FileUniqueID string `json:"file_unique_id"`
	Width        int    `json:"width"`
	Height       int    `json:"height"`
	FileSize     int64  `json:"file_size,omitempty"`
}

// sendMessageRequest is the payload POSTed to the sendMessage endpoint.
// See https://core.telegram.org/bots/api#sendmessage
type sendMessageRequest struct {
	ChatID           string `json:"chat_id"`
	Text             string `json:"text"`
	ReplyToMessageID int    `json:"reply_to_message_id,omitempty"`
}
