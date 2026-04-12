import type { Env, Platform, AdapterResult } from "./types";
import { SignatureError } from "./types";
import { publishToQueue } from "./queue";
import { verifySlack } from "./adapters/slack";
import { verifyDiscord } from "./adapters/discord";
import { verifyTelegram } from "./adapters/telegram";
import { verifyEmail } from "./adapters/email";
import { verifySms } from "./adapters/sms";
import { verifyWhatsapp } from "./adapters/whatsapp";
import { verifyTeams } from "./adapters/teams";
import { verifyWebchat } from "./adapters/webchat";

const adapters: Record<
  Platform,
  (req: Request, env: Env) => Promise<AdapterResult>
> = {
  slack: verifySlack,
  discord: verifyDiscord,
  telegram: verifyTelegram,
  email: verifyEmail,
  sms: verifySms,
  whatsapp: verifyWhatsapp,
  teams: verifyTeams,
  webchat: verifyWebchat,
};

export default {
  async fetch(request: Request, env: Env): Promise<Response> {
    if (request.method !== "POST") {
      return new Response("Method not allowed", { status: 405 });
    }

    const url = new URL(request.url);
    const segments = url.pathname.split("/").filter(Boolean);

    // Expected path: /webhooks/:platform
    if (segments.length < 2 || segments[0] !== "webhooks") {
      return new Response("Not found", { status: 404 });
    }

    const platform = segments[1] as Platform;
    const adapter = adapters[platform];
    if (!adapter) {
      return new Response(`Unknown platform: ${platform}`, { status: 404 });
    }

    try {
      const result = await adapter(request, env);
      await publishToQueue(env, result.message);
      return new Response("OK", { status: 200 });
    } catch (err) {
      if (err instanceof SignatureError) {
        return new Response("Unauthorized", { status: 401 });
      }
      console.error(`[${platform}] error:`, err);
      return new Response("Internal error", { status: 500 });
    }
  },
};

export { SignatureError };
