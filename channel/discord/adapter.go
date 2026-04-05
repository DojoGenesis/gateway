package discord

import (
	"context"
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/channel"
	"github.com/bwmarrin/discordgo"
)

// DiscordAdapter implements channel.WebhookAdapter for Discord Interactions
// delivered via HTTP webhook. It verifies Ed25519 signatures, normalizes
// Discord message payloads into channel.ChannelMessage, and sends outbound
// messages using a discordgo session.
type DiscordAdapter struct {
	cfg     DiscordConfig
	session *discordgo.Session // nil in tests that do not exercise Send
}

// New creates a DiscordAdapter from the provided config. If cfg.BotToken is
// non-empty a discordgo session is opened for outbound sends; callers that
// only need inbound handling (e.g. tests) may pass an empty BotToken.
func New(cfg DiscordConfig) (*DiscordAdapter, error) {
	a := &DiscordAdapter{cfg: cfg}

	if cfg.BotToken != "" {
		s, err := discordgo.New("Bot " + cfg.BotToken)
		if err != nil {
			return nil, fmt.Errorf("discord: failed to create session: %w", err)
		}
		a.session = s
	}

	return a, nil
}

// ---------------------------------------------------------------------------
// channel.ChannelAdapter
// ---------------------------------------------------------------------------

// Name satisfies channel.ChannelAdapter and returns "discord".
func (a *DiscordAdapter) Name() string { return "discord" }

// Capabilities returns the feature set supported by Discord.
func (a *DiscordAdapter) Capabilities() channel.AdapterCapabilities {
	return channel.AdapterCapabilities{
		SupportsThreads:     true,
		SupportsReactions:   true,
		SupportsAttachments: true,
		SupportsEdits:       true,
		MaxMessageLength:    2000,
	}
}

// Normalize converts a raw Discord message JSON payload into a
// channel.ChannelMessage. The raw bytes should be a Discord Message object
// (https://discord.com/developers/docs/resources/message).
func (a *DiscordAdapter) Normalize(raw []byte) (*channel.ChannelMessage, error) {
	var dm discordMessage
	if err := json.Unmarshal(raw, &dm); err != nil {
		return nil, fmt.Errorf("discord: normalize unmarshal: %w", err)
	}

	msg := &channel.ChannelMessage{
		ID:        dm.ID,
		Platform:  "discord",
		ChannelID: dm.ChannelID,
		UserID:    dm.Author.ID,
		UserName:  dm.Author.Username,
		Text:      dm.Content,
		Timestamp: dm.Timestamp,
	}

	// Replies carry a message_reference block.
	if dm.MessageReference != nil {
		msg.ThreadID = dm.MessageReference.MessageID
	}

	// Map Discord attachments to the normalized Attachment type.
	for _, a := range dm.Attachments {
		att := channel.Attachment{
			URL:      a.URL,
			Name:     a.Filename,
			Size:     int64(a.Size),
			MimeType: a.ContentType,
		}
		// Derive a coarse attachment type from the MIME type prefix.
		att.Type = attachmentType(a.ContentType)
		msg.Attachments = append(msg.Attachments, att)
	}

	return msg, nil
}

// Send delivers a channel.ChannelMessage to Discord via the discordgo session.
// If msg.ThreadID is non-empty the message is sent as a reply referencing
// that message ID. Returns an error if no session is available.
func (a *DiscordAdapter) Send(ctx context.Context, msg *channel.ChannelMessage) error {
	if a.session == nil {
		return fmt.Errorf("discord: no session available — provide a BotToken in DiscordConfig")
	}

	params := &discordgo.MessageSend{Content: msg.Text}

	if msg.ThreadID != "" {
		params.Reference = &discordgo.MessageReference{
			MessageID: msg.ThreadID,
			ChannelID: msg.ChannelID,
		}
	}

	_, err := a.session.ChannelMessageSendComplex(msg.ChannelID, params)
	if err != nil {
		return fmt.Errorf("discord: send failed: %w", err)
	}
	return nil
}

// ---------------------------------------------------------------------------
// channel.WebhookAdapter
// ---------------------------------------------------------------------------

// VerifySignature validates the Ed25519 signature Discord attaches to every
// interaction webhook. It reads X-Signature-Ed25519 and X-Signature-Timestamp
// and verifies them against the application public key stored in cfg.PublicKey.
//
// Discord requires the verified body to remain readable; this method does NOT
// consume r.Body — the caller (HandleWebhook) reads it after verification.
func (a *DiscordAdapter) VerifySignature(r *http.Request) error {
	sigHex := r.Header.Get("X-Signature-Ed25519")
	timestamp := r.Header.Get("X-Signature-Timestamp")

	if sigHex == "" || timestamp == "" {
		return fmt.Errorf("discord: missing signature headers")
	}

	pubKeyBytes, err := hex.DecodeString(a.cfg.PublicKey)
	if err != nil {
		return fmt.Errorf("discord: invalid public key hex: %w", err)
	}

	sig, err := hex.DecodeString(sigHex)
	if err != nil {
		return fmt.Errorf("discord: invalid signature hex: %w", err)
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		return fmt.Errorf("discord: failed to read request body: %w", err)
	}
	// Restore the body so downstream handlers can re-read it.
	r.Body = io.NopCloser(bytesReader(body))

	message := append([]byte(timestamp), body...)

	if !ed25519.Verify(ed25519.PublicKey(pubKeyBytes), message, sig) {
		return fmt.Errorf("discord: signature verification failed")
	}
	return nil
}

// HandleWebhook processes an inbound Discord Interactions webhook.
//
//   - Type 1 (PING): Discord sends this during endpoint registration; respond
//     immediately with {"type":1}.
//   - Type 2+ (APPLICATION_COMMAND / MESSAGE_COMPONENT / etc.): read, normalize,
//     and write 200 OK so Discord does not retry.
func (a *DiscordAdapter) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	if err := a.VerifySignature(r); err != nil {
		http.Error(w, "invalid signature", http.StatusUnauthorized)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "failed to read body", http.StatusInternalServerError)
		return
	}

	var interaction struct {
		Type int `json:"type"`
	}
	if err := json.Unmarshal(body, &interaction); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	// Type 1 = PING — Discord expects a synchronous type-1 ACK.
	if interaction.Type == 1 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"type":1}`))
		return
	}

	// All other interaction types: acknowledge with 200. In a production
	// system the body would be forwarded to the event bus for async handling.
	w.WriteHeader(http.StatusOK)
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

// discordMessage mirrors the subset of the Discord Message object we care
// about for normalization.
type discordMessage struct {
	ID        string    `json:"id"`
	ChannelID string    `json:"channel_id"`
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
	Author    struct {
		ID       string `json:"id"`
		Username string `json:"username"`
	} `json:"author"`
	MessageReference *struct {
		MessageID string `json:"message_id"`
	} `json:"message_reference,omitempty"`
	Attachments []struct {
		URL         string `json:"url"`
		Filename    string `json:"filename"`
		Size        int    `json:"size"`
		ContentType string `json:"content_type"`
	} `json:"attachments,omitempty"`
}

// attachmentType derives a coarse attachment category from a MIME type string.
func attachmentType(mime string) string {
	if len(mime) == 0 {
		return "file"
	}
	switch {
	case len(mime) >= 5 && mime[:5] == "image":
		return "image"
	case len(mime) >= 5 && mime[:5] == "video":
		return "video"
	case len(mime) >= 5 && mime[:5] == "audio":
		return "audio"
	default:
		return "file"
	}
}

// bytesReader wraps a byte slice as an io.Reader to restore r.Body after
// it has been consumed for signature verification.
func bytesReader(b []byte) io.Reader {
	return &byteSliceReader{b: b}
}

type byteSliceReader struct {
	b   []byte
	pos int
}

func (r *byteSliceReader) Read(p []byte) (int, error) {
	if r.pos >= len(r.b) {
		return 0, io.EOF
	}
	n := copy(p, r.b[r.pos:])
	r.pos += n
	return n, nil
}
