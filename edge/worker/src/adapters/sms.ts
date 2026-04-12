import type { Env, AdapterResult } from "../types";
import { SignatureError } from "../types";

export async function verifySms(
  req: Request,
  env: Env
): Promise<AdapterResult> {
  // Twilio HMAC-SHA1 signature verification
  // Twilio signs using: HMAC-SHA1(authToken, url + sorted_params)
  const twilioSignature = req.headers.get("x-twilio-signature") ?? "";
  if (!twilioSignature) {
    throw new SignatureError("sms", "missing x-twilio-signature header");
  }

  const body = await req.text();
  const url = req.url;

  // Parse form-encoded body (Twilio sends application/x-www-form-urlencoded)
  const params = new URLSearchParams(body);

  // Build sorted param string: sort keys, concatenate key+value pairs
  const sortedKeys = Array.from(params.keys()).sort();
  const paramString = sortedKeys.map((k) => `${k}${params.get(k) ?? ""}`).join("");
  const dataToSign = url + paramString;

  const key = await crypto.subtle.importKey(
    "raw",
    new TextEncoder().encode(env.DOJO_SMS_AUTH_TOKEN),
    { name: "HMAC", hash: "SHA-1" },
    false,
    ["sign"]
  );
  const signed = await crypto.subtle.sign(
    "HMAC",
    key,
    new TextEncoder().encode(dataToSign)
  );

  // Twilio sends the signature as base64
  const expectedBase64 = btoa(
    String.fromCharCode(...new Uint8Array(signed))
  );

  if (!timingSafeEqual(expectedBase64, twilioSignature)) {
    throw new SignatureError("sms");
  }

  return {
    message: {
      platform: "sms",
      channel_id: params.get("To") ?? "",
      sender_id: params.get("From") ?? "",
      sender_name: params.get("From") ?? "",
      content: params.get("Body") ?? "",
      timestamp: new Date().toISOString(),
      raw: Object.fromEntries(params.entries()),
      metadata: {
        message_sid: params.get("MessageSid") ?? "",
        num_media: params.get("NumMedia") ?? "0",
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
