import type { Env, AdapterResult } from "../types";
import { SignatureError } from "../types";

export async function verifyDiscord(
  req: Request,
  env: Env
): Promise<AdapterResult> {
  const body = await req.text();
  const signatureHex = req.headers.get("x-signature-ed25519") ?? "";
  const timestamp = req.headers.get("x-signature-timestamp") ?? "";

  if (!signatureHex || !timestamp) {
    throw new SignatureError("discord", "missing signature headers");
  }

  // Convert public key hex to CryptoKey
  const publicKeyBytes = hexToBytes(env.DOJO_DISCORD_PUBLIC_KEY);
  const publicKey = await crypto.subtle.importKey(
    "raw",
    publicKeyBytes,
    { name: "NODE-ED25519", namedCurve: "Ed25519" },
    false,
    ["verify"]
  );

  const message = new TextEncoder().encode(timestamp + body);
  const signatureBytes = hexToBytes(signatureHex);

  const valid = await crypto.subtle.verify(
    "NODE-ED25519",
    publicKey,
    signatureBytes,
    message
  );

  if (!valid) {
    throw new SignatureError("discord");
  }

  const payload = JSON.parse(body) as Record<string, unknown>;

  // Discord PING interaction (type 1) — must reply 200 with {type:1}
  // We emit a synthetic message and embed the ping flag in metadata so the
  // gateway consumer can short-circuit if needed.
  const isPing = payload.type === 1;

  const data = payload.data as Record<string, unknown> | undefined;
  return {
    message: {
      platform: "discord",
      channel_id: (payload.channel_id as string) ?? "",
      sender_id: (payload.member as Record<string, unknown>)?.user
        ? ((payload.member as Record<string, unknown>).user as Record<string, unknown>).id as string
        : (payload.user as Record<string, unknown>)?.id as string ?? "",
      sender_name: "",
      content: (data?.options as Array<Record<string, unknown>>)?.[0]?.value as string ?? "",
      timestamp: new Date().toISOString(),
      raw: payload,
      metadata: isPing ? { discord_ping: "1" } : undefined,
    },
  };
}

function hexToBytes(hex: string): Uint8Array {
  if (hex.length % 2 !== 0) {
    throw new Error("invalid hex string");
  }
  const bytes = new Uint8Array(hex.length / 2);
  for (let i = 0; i < hex.length; i += 2) {
    bytes[i / 2] = parseInt(hex.slice(i, i + 2), 16);
  }
  return bytes;
}
