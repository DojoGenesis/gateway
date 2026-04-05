package whatsapp

// WebhookPayload is the top-level payload sent by the WhatsApp Cloud API
// to a registered webhook endpoint.
// See https://developers.facebook.com/docs/whatsapp/cloud-api/webhooks/payload-examples
type WebhookPayload struct {
	Object string  `json:"object"`
	Entry  []Entry `json:"entry"`
}

// Entry represents a single business account entry in a webhook payload.
type Entry struct {
	ID      string   `json:"id"`
	Changes []Change `json:"changes"`
}

// Change represents a single field change within an entry.
type Change struct {
	Value ChangeValue `json:"value"`
	Field string      `json:"field"`
}

// ChangeValue holds the message and contact data for a change event.
type ChangeValue struct {
	MessagingProduct string    `json:"messaging_product"`
	Metadata         Metadata  `json:"metadata"`
	Contacts         []Contact `json:"contacts,omitempty"`
	Messages         []Message `json:"messages,omitempty"`
}

// Metadata holds information about the receiving phone number.
type Metadata struct {
	DisplayPhoneNumber string `json:"display_phone_number"`
	PhoneNumberID      string `json:"phone_number_id"`
}

// Contact represents a WhatsApp contact associated with a message.
type Contact struct {
	Profile ContactProfile `json:"profile"`
	WaID    string         `json:"wa_id"`
}

// ContactProfile holds the contact's display name.
type ContactProfile struct {
	Name string `json:"name"`
}

// Message represents an inbound WhatsApp message.
// See https://developers.facebook.com/docs/whatsapp/cloud-api/webhooks/components#messages-object
type Message struct {
	// From is the sender's WhatsApp phone number (in E.164 format without +).
	From string `json:"from"`

	// ID is the WhatsApp message ID.
	ID string `json:"id"`

	// Timestamp is the Unix timestamp of the message as a string.
	Timestamp string `json:"timestamp"`

	// Type indicates the message type: "text", "image", "document", etc.
	Type string `json:"type"`

	// Text is set when Type == "text".
	Text *TextBody `json:"text,omitempty"`

	// Image is set when Type == "image".
	Image *Media `json:"image,omitempty"`

	// Document is set when Type == "document".
	Document *Media `json:"document,omitempty"`

	// Context is set when the message is a reply to another message.
	Context *MessageContext `json:"context,omitempty"`
}

// TextBody holds the body of a text message.
type TextBody struct {
	Body string `json:"body"`
}

// Media holds media attachment metadata.
type Media struct {
	// ID is the WhatsApp media object ID (resolve via the media API).
	ID string `json:"id"`

	// MimeType is the MIME content type.
	MimeType string `json:"mime_type,omitempty"`

	// SHA256 is the SHA-256 hash of the media file.
	SHA256 string `json:"sha256,omitempty"`

	// Caption is the optional caption for the media.
	Caption string `json:"caption,omitempty"`

	// Filename is the filename for document attachments.
	Filename string `json:"filename,omitempty"`
}

// MessageContext is set when a message is a reply to another message.
type MessageContext struct {
	// From is the WhatsApp ID of the message being replied to.
	From string `json:"from"`

	// ID is the message ID being replied to.
	ID string `json:"id"`
}

// sendMessageRequest is the JSON body for POST /{phone-number-id}/messages.
// See https://developers.facebook.com/docs/whatsapp/cloud-api/messages/text-messages
type sendMessageRequest struct {
	MessagingProduct string       `json:"messaging_product"`
	To               string       `json:"to"`
	Type             string       `json:"type"`
	Text             *sendTextBody `json:"text,omitempty"`
}

// sendTextBody is the text payload within a send request.
type sendTextBody struct {
	Body string `json:"body"`
}
