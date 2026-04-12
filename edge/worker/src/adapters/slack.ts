import type { Env, AdapterResult } from "../types";
import { SignatureError } from "../types";

export async function verifySlack(
  req: Request,
  env: Env
): Promise<AdapterResult> {
  const body = await req.text();
  const timestamp = req.headers.get("x-slack-request-timestamp") ?? "";
  const signature = req.headers.get("x-slack-signature") ?? "";

  // Prevent replay attacks (5 min window)
  const now = Math.floor(Date.now() / 1000);
  if (Math.abs(now - parseInt(timestamp, 10)) > 300) {
    throw new SignatureError("slack", "timestamp too old");
  }

  const sigBasestring = `v0:${timestamp}:${body}`;
  const key = await crypto.subtle.importKey(
    "raw",
    new TextEncoder().encode(env.DOJO_SLACK_SIGNINGSECRET),
    { name: "HMAC", hash: "SHA-256" },
    false,
    ["sign"]
  );
  const signed = await crypto.subtle.sign(
    "HMAC",
    key,
    new TextEncoder().encode(sigBasestring)
  );
  const expected =
    "v0=" +
    Array.from(new Uint8Array(signed))
      .map((b) => b.toString(16).padStart(2, "0"))
      .join("");

  if (!timingSafeEqual(expected, signature)) {
    throw new SignatureError("slack");
  }

  const payload = JSON.parse(body) as Record<string, unknown>;

  // Slack URL verification challenge (one-time during app setup)
  const challenge = (payload as { challenge?: string }).challenge;
  if (challenge) {
    // Caller will receive "OK" 200 — but the challenge response needs to be the
    // challenge value. Surface it via a thrown marker so the router can handle it.
    // Instead, we return a synthetic message and the router returns 200 "OK".
    // The Slack URL verification flow expects the challenge echoed back in JSON.
    // We handle this as a special case by embedding a marker in metadata.
  }

  const event = payload.event as Record<string, unknown> | undefined;
  return {
    message: {
      platform: "slack",
      channel_id: (event?.channel as string) ?? "",
      sender_id: (event?.user as string) ?? "",
      sender_name: "",
      content: (event?.text as string) ?? "",
      thread_id: event?.thread_ts as string | undefined,
      timestamp: new Date().toISOString(),
      raw: payload,
      metadata: challenge ? { slack_challenge: challenge } : undefined,
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
