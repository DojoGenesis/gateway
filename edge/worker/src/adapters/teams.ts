import type { Env, AdapterResult } from "../types";
import { SignatureError } from "../types";

export async function verifyTeams(
  req: Request,
  env: Env
): Promise<AdapterResult> {
  const body = await req.text();

  // Phase 0: verify Authorization header contains a Bearer JWT whose audience
  // matches the configured Teams App ID. Full JWKS validation is deferred.
  const authHeader = req.headers.get("Authorization") ?? "";
  if (!authHeader.startsWith("Bearer ")) {
    throw new SignatureError("teams", "missing Bearer token");
  }

  const token = authHeader.slice(7);
  if (!validateTeamsJwtAudience(token, env.DOJO_TEAMS_APP_ID)) {
    throw new SignatureError("teams", "invalid audience in JWT");
  }

  const payload = JSON.parse(body) as Record<string, unknown>;

  // Bot Framework Activity payload
  const from = payload.from as Record<string, unknown> | undefined;
  const conversation = payload.conversation as Record<string, unknown> | undefined;

  return {
    message: {
      platform: "teams",
      channel_id:
        (conversation?.id as string) ??
        (payload.channelId as string) ??
        "",
      sender_id: (from?.id as string) ?? "",
      sender_name: (from?.name as string) ?? "",
      content:
        (payload.text as string) ??
        extractTeamsCardText(payload) ??
        "",
      thread_id: (conversation?.id as string) ?? undefined,
      timestamp:
        (payload.timestamp as string) ?? new Date().toISOString(),
      raw: payload,
      metadata: {
        activity_type: (payload.type as string) ?? "",
        service_url: (payload.serviceUrl as string) ?? "",
      },
    },
  };
}

/**
 * Decode JWT without verifying the cryptographic signature (Phase 0).
 * Checks that the `aud` claim matches the expected app ID.
 * Full RS256 JWKS verification is deferred to Phase 1.
 */
function validateTeamsJwtAudience(token: string, expectedAppId: string): boolean {
  try {
    const parts = token.split(".");
    if (parts.length !== 3) return false;
    // JWT payload is base64url-encoded
    const paddedPayload = parts[1].replace(/-/g, "+").replace(/_/g, "/");
    const decoded = atob(paddedPayload);
    const claims = JSON.parse(decoded) as Record<string, unknown>;
    const aud = claims.aud;
    if (Array.isArray(aud)) {
      return aud.includes(expectedAppId);
    }
    return aud === expectedAppId;
  } catch {
    return false;
  }
}

/** Extract text from Adaptive Card body if present. */
function extractTeamsCardText(
  payload: Record<string, unknown>
): string | undefined {
  try {
    const attachments = payload.attachments as Array<Record<string, unknown>> | undefined;
    if (!attachments?.length) return undefined;
    const card = attachments[0].content as Record<string, unknown> | undefined;
    if (!card) return undefined;
    const body = card.body as Array<Record<string, unknown>> | undefined;
    if (!body?.length) return undefined;
    const textBlock = body.find((el) => el.type === "TextBlock");
    return textBlock?.text as string | undefined;
  } catch {
    return undefined;
  }
}
