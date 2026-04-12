import type { ChannelMessage, Env } from "./types";

export async function publishToQueue(
  env: Env,
  message: ChannelMessage
): Promise<void> {
  await env.DOJO_CHANNEL_EVENTS.send(message);
}
