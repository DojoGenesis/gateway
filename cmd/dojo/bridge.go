// Package main provides the bridge subcommand that wires ChannelBridge to the
// runtime/event NATS bus. This is the production entry point for channel traffic
// — InProcessBus is NOT used here (Era 3 Phase 1 Track A).
//
// Phase 2 (2026-04-09): all 8 platform adapters registered; HTTP server listens
// on DOJO_BRIDGE_PORT (default 8090) to receive inbound webhooks.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/DojoGenesis/gateway/channel"
	"github.com/DojoGenesis/gateway/channel/discord"
	"github.com/DojoGenesis/gateway/channel/email"
	"github.com/DojoGenesis/gateway/channel/slack"
	"github.com/DojoGenesis/gateway/channel/sms"
	"github.com/DojoGenesis/gateway/channel/teams"
	"github.com/DojoGenesis/gateway/channel/telegram"
	"github.com/DojoGenesis/gateway/channel/webchat"
	"github.com/DojoGenesis/gateway/channel/whatsapp"
	"github.com/DojoGenesis/gateway/runtime/event"
)

// NATSBusAdapter bridges the runtime/event.Bus to the channel.NATSPublisher
// and channel.NATSSubscriber interfaces. This is the thin adapter referenced
// in channel/nats_bus.go that the wiring layer (cmd/dojo) provides.
type NATSBusAdapter struct {
	bus event.Bus
}

// NewNATSBusAdapter wraps a runtime/event.Bus for use with channel.NATSBus.
func NewNATSBusAdapter(bus event.Bus) *NATSBusAdapter {
	return &NATSBusAdapter{bus: bus}
}

// PublishRaw publishes raw event bytes on the given subject via the NATS bus.
// It wraps the bytes in an event.Event and publishes through the bus.
func (a *NATSBusAdapter) PublishRaw(ctx context.Context, subject string, data []byte) error {
	evt := event.Event{
		ID:   subject,
		Type: subject,
		Data: data,
	}
	return a.bus.Publish(ctx, evt)
}

// SubscribeRaw subscribes to events on the given subject pattern and delivers
// raw bytes to the handler. Returns an unsubscribe function.
func (a *NATSBusAdapter) SubscribeRaw(ctx context.Context, subjectPattern string, handler func(string, []byte)) (func(), error) {
	sub, err := a.bus.Subscribe(ctx, event.EventFilter{
		Types: []string{subjectPattern},
	}, func(_ context.Context, evt event.Event) error {
		handler(evt.Type, evt.Data)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return func() { sub.Unsubscribe() }, nil
}

// runBridgeCommand starts the Channel Bridge with NATS bus wiring and an HTTP
// server to receive inbound webhooks from all registered platform adapters.
func runBridgeCommand(args []string) error {
	slog.Info("bridge: starting Channel Bridge with NATS bus")

	// Parse optional config path.
	_ = args // reserved for future --config flag

	// Create embedded NATS bus with JetStream enabled.
	cfg := event.DefaultConfig()
	cfg.Enabled = true
	cfg.WAL.Enabled = true

	dataDir := os.Getenv("DOJO_DATA_DIR")
	if dataDir != "" {
		cfg.WAL.DBPath = dataDir + "/event-wal.db"
	}

	bus, err := event.NewBus(cfg)
	if err != nil {
		return fmt.Errorf("bridge: create NATS bus: %w", err)
	}
	defer bus.Close()

	slog.Info("bridge: NATS bus created with JetStream")

	// Create the NATSBus adapter for the channel module.
	adapter := NewNATSBusAdapter(bus)
	natsBus := channel.NewNATSBus(adapter, channel.WithNATSSubscriber(adapter))

	// Determine credential store backend.
	credStore := resolveCredentialStore()

	// Create the adapter registry with credential store injection.
	registry := channel.NewAdapterRegistry(credStore)

	// Create the WebhookGateway backed by NATSBus (not InProcessBus).
	gw := channel.NewWebhookGateway(natsBus, credStore)

	// Register all platform adapters. Only adapters whose required credentials
	// are present in the credential store are registered; others are skipped with a WARN log.
	buildAndRegisterAdapters(gw, credStore)

	// Create the bridge with nil runner (runner will be injected when
	// workflow execution is wired in Phase 3).
	bridge := channel.NewChannelBridge(nil)

	// Subscribe the bridge to the NATS bus.
	natsBus.Subscribe(bridge.BusHandler(context.Background()))

	// Subscribe to all channel events from NATS.
	if err := natsBus.SubscribeNATS(context.Background(), channel.ChannelSubjectWildcard()); err != nil {
		return fmt.Errorf("bridge: subscribe to NATS channel events: %w", err)
	}

	slog.Info("bridge: Channel Bridge wired",
		"credential_backend", credentialBackendName(),
		"adapters", registry.List(),
	)

	// Start the HTTP server to receive inbound webhooks.
	port := bridgePort()
	mux := http.NewServeMux()
	mux.Handle("/webhooks/", gw)
	srv := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	go func() {
		slog.Info("bridge: HTTP server listening", "addr", ":"+port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("bridge: HTTP server error", "error", err)
		}
	}()

	// Wait for shutdown signal.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	<-ctx.Done()
	slog.Info("bridge: shutting down")

	if err := srv.Shutdown(context.Background()); err != nil {
		slog.Warn("bridge: HTTP server shutdown error", "error", err)
	}
	natsBus.Close()
	return nil
}

// buildAndRegisterAdapters constructs and registers all platform adapters with
// the WebhookGateway. Credentials are fetched from the CredentialStore (which
// may be EnvCredentialStore or InfisicalCredentialStore depending on
// DOJO_CREDENTIAL_BACKEND). Adapters with missing credentials are skipped with
// a WARN log — nothing crashes.
func buildAndRegisterAdapters(gw *channel.WebhookGateway, creds channel.CredentialStore) {
	ctx := context.Background()

	// cred fetches a credential from the store, returning "" on any error.
	cred := func(platform, key string) string {
		v, err := creds.Get(ctx, platform, key)
		if err != nil {
			return ""
		}
		return v
	}

	// --- Phase 1 adapters ---

	// Slack: requires TOKEN + SIGNINGSECRET
	slackToken := cred("slack", "TOKEN")
	slackSecret := cred("slack", "SIGNINGSECRET")
	if slackToken != "" && slackSecret != "" {
		gw.Register("slack", slack.New(slack.SlackConfig{
			BotToken:      slackToken,
			SigningSecret: slackSecret,
			Mode:          envOr("DOJO_SLACK_MODE", "http"),
			AppToken:      cred("slack", "APPTOKEN"),
		}))
		slog.Info("bridge: registered adapter", "platform", "slack")
	} else {
		slog.Warn("bridge: skipping adapter (missing credentials)", "platform", "slack",
			"hint", "set DOJO_SLACK_TOKEN and DOJO_SLACK_SIGNINGSECRET")
	}

	// Discord: requires BOT_TOKEN + PUBLIC_KEY
	discordToken := cred("discord", "BOT_TOKEN")
	discordPublicKey := cred("discord", "PUBLIC_KEY")
	if discordToken != "" && discordPublicKey != "" {
		discordAdapter, err := discord.New(discord.DiscordConfig{
			BotToken:  discordToken,
			PublicKey: discordPublicKey,
			AppID:     cred("discord", "APP_ID"),
			GuildID:   cred("discord", "GUILD_ID"),
		})
		if err != nil {
			slog.Error("bridge: failed to create Discord adapter", "error", err)
		} else {
			gw.Register("discord", discordAdapter)
			slog.Info("bridge: registered adapter", "platform", "discord")
		}
	} else {
		slog.Warn("bridge: skipping adapter (missing credentials)", "platform", "discord",
			"hint", "set DOJO_DISCORD_BOT_TOKEN and DOJO_DISCORD_PUBLIC_KEY")
	}

	// Telegram: requires BOT_TOKEN
	telegramToken := cred("telegram", "BOT_TOKEN")
	if telegramToken != "" {
		gw.Register("telegram", telegram.NewTelegramAdapter(
			telegramToken,
			cred("telegram", "SECRET_TOKEN"),
		))
		slog.Info("bridge: registered adapter", "platform", "telegram")
	} else {
		slog.Warn("bridge: skipping adapter (missing credentials)", "platform", "telegram",
			"hint", "set DOJO_TELEGRAM_BOT_TOKEN")
	}

	// --- Phase 2 adapters ---

	// Email (SendGrid Inbound Parse): requires WEBHOOK_SECRET + SENDGRID_API_KEY
	emailSecret := cred("email", "WEBHOOK_SECRET")
	emailAPIKey := cred("email", "SENDGRID_API_KEY")
	if emailSecret != "" && emailAPIKey != "" {
		gw.Register("email", email.New(email.EmailConfig{
			WebhookSecret:  emailSecret,
			SendGridAPIKey: emailAPIKey,
			FromAddress:    envOr("DOJO_EMAIL_FROM_ADDRESS", "noreply@dojo.example"),
			FromName:       envOr("DOJO_EMAIL_FROM_NAME", "Dojo"),
		}))
		slog.Info("bridge: registered adapter", "platform", "email")
	} else {
		slog.Warn("bridge: skipping adapter (missing credentials)", "platform", "email",
			"hint", "set DOJO_EMAIL_WEBHOOK_SECRET and DOJO_EMAIL_SENDGRID_API_KEY")
	}

	// SMS (Twilio): requires ACCOUNT_SID + AUTH_TOKEN
	smsAccountSID := cred("sms", "ACCOUNT_SID")
	smsAuthToken := cred("sms", "AUTH_TOKEN")
	if smsAccountSID != "" && smsAuthToken != "" {
		gw.Register("sms", sms.NewSMSAdapter(sms.SMSConfig{
			AccountSID: smsAccountSID,
			AuthToken:  smsAuthToken,
			FromNumber: cred("sms", "FROM_NUMBER"),
		}))
		slog.Info("bridge: registered adapter", "platform", "sms")
	} else {
		slog.Warn("bridge: skipping adapter (missing credentials)", "platform", "sms",
			"hint", "set DOJO_SMS_ACCOUNT_SID and DOJO_SMS_AUTH_TOKEN")
	}

	// WhatsApp (Meta Cloud API): requires PHONE_NUMBER_ID + ACCESS_TOKEN
	waPhoneID := cred("whatsapp", "PHONE_NUMBER_ID")
	waAccessToken := cred("whatsapp", "ACCESS_TOKEN")
	if waPhoneID != "" && waAccessToken != "" {
		gw.Register("whatsapp", whatsapp.NewWhatsAppAdapter(whatsapp.WhatsAppConfig{
			PhoneNumberID: waPhoneID,
			AccessToken:   waAccessToken,
			VerifyToken:   cred("whatsapp", "VERIFY_TOKEN"),
			AppSecret:     cred("whatsapp", "APP_SECRET"),
		}))
		slog.Info("bridge: registered adapter", "platform", "whatsapp")
	} else {
		slog.Warn("bridge: skipping adapter (missing credentials)", "platform", "whatsapp",
			"hint", "set DOJO_WHATSAPP_PHONE_NUMBER_ID and DOJO_WHATSAPP_ACCESS_TOKEN")
	}

	// Teams (Microsoft Bot Framework): requires BOT_TOKEN + APP_ID
	teamsBotToken := cred("teams", "BOT_TOKEN")
	teamsAppID := cred("teams", "APP_ID")
	if teamsBotToken != "" && teamsAppID != "" {
		gw.Register("teams", teams.NewTeamsAdapter(teamsBotToken, teamsAppID))
		slog.Info("bridge: registered adapter", "platform", "teams")
	} else {
		slog.Warn("bridge: skipping adapter (missing credentials)", "platform", "teams",
			"hint", "set DOJO_TEAMS_BOT_TOKEN and DOJO_TEAMS_APP_ID")
	}

	// WebChat (embedded widget): TOKEN is optional — always registered.
	gw.Register("webchat", webchat.NewWebChatAdapter(cred("webchat", "TOKEN")))
	slog.Info("bridge: registered adapter", "platform", "webchat")
}

// bridgePort returns the HTTP port for the bridge server.
// Defaults to 8090 to avoid collision with the main Gateway (typically 8080).
func bridgePort() string {
	if p := os.Getenv("DOJO_BRIDGE_PORT"); p != "" {
		return p
	}
	return "8090"
}

// envOr returns the value of the environment variable key, or fallback if unset.
func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// resolveCredentialStore returns the appropriate CredentialStore based on
// the DOJO_CREDENTIAL_BACKEND environment variable.
// Supported values: "env" (default), "infisical".
//
// When DOJO_CREDENTIAL_BACKEND=infisical, the following env vars are required:
//
//	DOJO_INFISICAL_CLIENT_ID       — machine identity client ID
//	DOJO_INFISICAL_CLIENT_SECRET   — machine identity client secret
//	DOJO_INFISICAL_PROJECT_ID      — Infisical project ID
//
// Optional:
//
//	DOJO_INFISICAL_SITE_URL        — self-hosted URL (default: https://app.infisical.com)
//	DOJO_INFISICAL_ENVIRONMENT     — environment slug (default: prod)
//	DOJO_INFISICAL_SECRET_PATH     — path prefix (default: /channel)
func resolveCredentialStore() channel.CredentialStore {
	backend := os.Getenv("DOJO_CREDENTIAL_BACKEND")
	switch backend {
	case "infisical":
		clientID := os.Getenv("DOJO_INFISICAL_CLIENT_ID")
		clientSecret := os.Getenv("DOJO_INFISICAL_CLIENT_SECRET")
		projectID := os.Getenv("DOJO_INFISICAL_PROJECT_ID")

		if clientID == "" || clientSecret == "" || projectID == "" {
			slog.Warn("bridge: DOJO_INFISICAL_CLIENT_ID / CLIENT_SECRET / PROJECT_ID not set, falling back to env store")
			return channel.NewEnvCredentialStore()
		}

		cfg := channel.InfisicalConfig{
			SiteURL:      envOr("DOJO_INFISICAL_SITE_URL", "https://app.infisical.com"),
			ClientID:     clientID,
			ClientSecret: clientSecret,
			ProjectID:    projectID,
			Environment:  envOr("DOJO_INFISICAL_ENVIRONMENT", "prod"),
			SecretPath:   envOr("DOJO_INFISICAL_SECRET_PATH", "/channel"),
		}

		httpClient := channel.NewInfisicalHTTPClient(cfg.SiteURL, cfg.ClientID, cfg.ClientSecret, cfg.ProjectID)
		slog.Info("bridge: using Infisical credential store",
			"site", cfg.SiteURL,
			"project", cfg.ProjectID,
			"environment", cfg.Environment,
			"secret_path", cfg.SecretPath,
		)
		return channel.NewInfisicalCredentialStore(httpClient, cfg)
	default:
		return channel.NewEnvCredentialStore()
	}
}

// credentialBackendName returns the name of the active credential backend.
func credentialBackendName() string {
	backend := os.Getenv("DOJO_CREDENTIAL_BACKEND")
	if backend == "" {
		return "env"
	}
	return backend
}
