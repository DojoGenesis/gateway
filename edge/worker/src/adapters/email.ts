import type { Env, AdapterResult } from "../types";
import { SignatureError } from "../types";

export async function verifyEmail(
  req: Request,
  env: Env
): Promise<AdapterResult> {
  const body = await req.text();

  // HMAC-SHA256 verification — signature delivered via x-webhook-signature header
  const signature = req.headers.get("x-webhook-signature") ?? "";
  if (!signature) {
    throw new SignatureError("email", "missing signature header");
  }

  const key = await crypto.subtle.importKey(
    "raw",
    new TextEncoder().encode(env.DOJO_EMAIL_WEBHOOK_SECRET),
    { name: "HMAC", hash: "SHA-256" },
    false,
    ["sign"]
  );
  const signed = await crypto.subtle.sign(
    "HMAC",
    key,
    new TextEncoder().encode(body)
  );
  const expected = Array.from(new Uint8Array(signed))
    .map((b) => b.toString(16).padStart(2, "0"))
    .join("");

  // Strip any prefix like "sha256=" if present
  const normalizedSignature = signature.startsWith("sha256=")
    ? signature.slice(7)
    : signature;

  if (!timingSafeEqual(expected, normalizedSignature)) {
    throw new SignatureError("email");
  }

  const payload = JSON.parse(body) as Record<string, unknown>;

  // Normalize common email webhook payload fields
  // Compatible with SendGrid, Mailgun, Postmark event formats
  const from = (payload.from as string) ?? (payload.sender as string) ?? "";
  const subject = (payload.subject as string) ?? "";
  const text = (payload.text as string) ?? (payload.body_plain as string) ?? (payload.TextBody as string) ?? "";
  const messageId = (payload.message_id as string) ?? (payload.MessageID as string) ?? "";

  return {
    message: {
      platform: "email",
      channel_id: (payload.to as string) ?? (payload.recipient as string) ?? "",
      sender_id: from,
      sender_name: (payload.from_name as string) ?? from,
      content: subject ? `[${subject}] ${text}` : text,
      thread_id: (payload.in_reply_to as string) ?? undefined,
      timestamp: new Date().toISOString(),
      raw: payload,
      metadata: messageId ? { message_id: messageId } : undefined,
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
