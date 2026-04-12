/** ChannelMessage mirrors the Go ChannelMessage struct from the Gateway. */
export interface ChannelMessage {
  platform: Platform;
  channel_id: string;
  sender_id: string;
  sender_name: string;
  content: string;
  thread_id?: string;
  timestamp: string;
  raw: Record<string, unknown>;
  metadata?: Record<string, string>;
}

export type Platform =
  | "slack"
  | "discord"
  | "telegram"
  | "email"
  | "sms"
  | "whatsapp"
  | "teams"
  | "webchat";

export interface Env {
  DOJO_CAS: D1Database;
  DOJO_CHANNEL_EVENTS: Queue<ChannelMessage>;
  DOJO_SLACK_SIGNINGSECRET: string;
  DOJO_DISCORD_PUBLIC_KEY: string;
  DOJO_TELEGRAM_SECRET_TOKEN: string;
  DOJO_EMAIL_WEBHOOK_SECRET: string;
  DOJO_SMS_AUTH_TOKEN: string;
  DOJO_WHATSAPP_APP_SECRET: string;
  DOJO_TEAMS_APP_ID: string;
  DOJO_WEBCHAT_TOKEN: string;
}

export interface AdapterResult {
  message: ChannelMessage;
}

export class SignatureError extends Error {
  constructor(platform: string, detail?: string) {
    super(
      `Signature verification failed for ${platform}${detail ? ": " + detail : ""}`
    );
    this.name = "SignatureError";
  }
}
