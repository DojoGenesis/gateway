import type { Env, AdapterResult } from "../types";
import { SignatureError } from "../types";

export async function verifyWebchat(
  req: Request,
  env: Env
): Promise<AdapterResult> {
  const body = await req.text();

  // Simple Bearer token check — webchat clients embed the token in Authorization
  const authHeader = req.headers.get("Authorization") ?? "";
  if (!authHeader.startsWith("Bearer ")) {
    throw new SignatureError("webchat", "missing Bearer token");
  }

  const token = authHeader.slice(7);
  if (!timingSafeEqual(token, env.DOJO_WEBCHAT_TOKEN)) {
    throw new SignatureError("webchat", "invalid token");
  }

  const payload = JSON.parse(body) as Record<string, unknown>;

  // Webchat payload is defined by the first-party widget — fields are explicit
  const sessionId = (payload.session_id as string) ?? "";
  const userId = (payload.user_id as string) ?? (payload.visitor_id as string) ?? sessionId;
  const userName = (payload.user_name as string) ?? (payload.name as string) ?? "";
  const content = (payload.message as string) ?? (payload.text as string) ?? "";
  const threadId = (payload.thread_id as string) ?? undefined;
  const page = (payload.page as string) ?? (payload.url as string) ?? "";

  return {
    message: {
      platform: "webchat",
      channel_id: sessionId,
      sender_id: userId,
      sender_name: userName,
      content,
      thread_id: threadId,
      timestamp: (payload.timestamp as string) ?? new Date().toISOString(),
      raw: payload,
      metadata: page ? { page_url: page } : undefined,
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
