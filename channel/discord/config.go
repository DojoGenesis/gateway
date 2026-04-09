// Package discord implements a WebhookAdapter for the Discord platform.
// It handles Discord Interactions (slash commands, components) delivered via
// HTTP webhook and wraps discordgo for outbound message delivery.
package discord

// DiscordConfig holds the credentials and identifiers required to operate
// the Discord adapter. Values are typically sourced from environment variables
// or a CredentialStore rather than hard-coded.
type DiscordConfig struct {
	// BotToken is the Discord bot token used to authenticate outbound API
	// calls via discordgo (e.g. "Bot MTIz...").
	BotToken string

	// PublicKey is the application's Ed25519 public key (hex-encoded) used to
	// verify the X-Signature-Ed25519 header on inbound interaction webhooks.
	PublicKey string

	// AppID is the Discord application/client ID.
	AppID string

	// GuildID is the Discord guild (server) ID for per-guild resume state.
	// When empty, resume state is stored with the key "discord.resume.default".
	GuildID string
}

// ResumeStateKey returns the NATS KV key for Opcode 6 resume state.
// Format: discord.resume.{guild_id}. TTL managed by the KV bucket (7d).
func (c DiscordConfig) ResumeStateKey() string {
	gid := c.GuildID
	if gid == "" {
		gid = "default"
	}
	return "discord.resume." + gid
}
