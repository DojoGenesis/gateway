import type { Env, AdapterResult } from "../types";
import { SignatureError } from "../types";

export async function verifyTelegram(
  req: Request,
  env: Env
): Promise<AdapterResult> {
  const body = await req.text();

  // Telegram sends a secret token in the header when configured via
  // setWebhook with the secret_token parameter.
  const secretToken = req.headers.get("x-telegram-bot-api-secret-token") ?? "";
  if (!timingSafeEqual(secretToken, env.DOJO_TELEGRAM_SECRET_TOKEN)) {
    throw new SignatureError("telegram", "invalid secret token");
  }

  const payload = JSON.parse(body) as Record<string, unknown>;

  // Extract message details — Telegram update objects vary by type
  const message = payload.message as Record<string, unknown> | undefined;
  const editedMessage = payload.edited_message as Record<string, unknown> | undefined;
  const msg = message ?? editedMessage;

  const from = msg?.from as Record<string, unknown> | undefined;
  const chat = msg?.chat as Record<string, unknown> | undefined;
  const firstName = (from?.first_name as string) ?? "";
  const lastName = (from?.last_name as string) ?? "";
  const senderName = lastName ? `${firstName} ${lastName}`.trim() : firstName;

  return {
    message: {
      platform: "telegram",
      channel_id: String(chat?.id ?? ""),
      sender_id: String(from?.id ?? ""),
      sender_name: senderName,
      content: (msg?.text as string) ?? "",
      thread_id: msg?.message_thread_id !== undefined
        ? String(msg.message_thread_id)
        : undefined,
      timestamp: msg?.date
        ? new Date((msg.date as number) * 1000).toISOString()
        : new Date().toISOString(),
      raw: payload,
    },
  };
}

function timingSafeEqual(a: string, b: string): boolean {
  if (a.length !== b.length) return false;
  const encoder = new TextEncoder();
  const ab = encoder.encode(a);
  const bb = encoder.encode(b);
  let result = 0;
  for (let i = 0; i < ab.length; i++) {
    result |= ab[i] ^ bb[i];
  }
  return result === 0;
}
