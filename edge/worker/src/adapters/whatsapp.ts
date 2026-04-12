import type { Env, AdapterResult } from "../types";
import { SignatureError } from "../types";

export async function verifyWhatsapp(
  req: Request,
  env: Env
): Promise<AdapterResult> {
  const body = await req.text();

  // Meta HMAC-SHA256 verification via x-hub-signature-256 header
  const hubSignature = req.headers.get("x-hub-signature-256") ?? "";
  if (!hubSignature) {
    throw new SignatureError("whatsapp", "missing x-hub-signature-256 header");
  }

  const key = await crypto.subtle.importKey(
    "raw",
    new TextEncoder().encode(env.DOJO_WHATSAPP_APP_SECRET),
    { name: "HMAC", hash: "SHA-256" },
    false,
    ["sign"]
  );
  const signed = await crypto.subtle.sign(
    "HMAC",
    key,
    new TextEncoder().encode(body)
  );
  const expectedHex = Array.from(new Uint8Array(signed))
    .map((b) => b.toString(16).padStart(2, "0"))
    .join("");
  const expected = `sha256=${expectedHex}`;

  if (!timingSafeEqual(expected, hubSignature)) {
    throw new SignatureError("whatsapp");
  }

  const payload = JSON.parse(body) as Record<string, unknown>;

  // WhatsApp Cloud API payload structure
  const entry = (payload.entry as Array<Record<string, unknown>>)?.[0];
  const changes = (entry?.changes as Array<Record<string, unknown>>)?.[0];
  const value = changes?.value as Record<string, unknown> | undefined;
  const messages = value?.messages as Array<Record<string, unknown>> | undefined;
  const msg = messages?.[0];
  const contacts = value?.contacts as Array<Record<string, unknown>> | undefined;
  const contact = contacts?.[0];

  const senderId = (msg?.from as string) ?? "";
  const senderName = (contact?.profile as Record<string, unknown>)?.name as string ?? senderId;
  const phoneNumberId = (value?.metadata as Record<string, unknown>)?.phone_number_id as string ?? "";

  // Extract text content — handle type:text and type:interactive
  let content = "";
  if (msg?.type === "text") {
    content = (msg.text as Record<string, unknown>)?.body as string ?? "";
  } else if (msg?.type === "interactive") {
    const interactive = msg.interactive as Record<string, unknown>;
    content =
      (interactive?.button_reply as Record<string, unknown>)?.title as string ??
      (interactive?.list_reply as Record<string, unknown>)?.title as string ??
      "";
  }

  return {
    message: {
      platform: "whatsapp",
      channel_id: phoneNumberId,
      sender_id: senderId,
      sender_name: senderName,
      content,
      timestamp: msg?.timestamp
        ? new Date((msg.timestamp as number) * 1000).toISOString()
        : new Date().toISOString(),
      raw: payload,
      metadata: {
        message_id: (msg?.id as string) ?? "",
        message_type: (msg?.type as string) ?? "",
      },
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
