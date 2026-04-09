# ADR-015 Addendum: Channel NATS Subject Conventions

**Date:** 2026-04-09
**Status:** ACCEPTED
**Amends:** ADR-015 (Embedded NATS Event Bus)
**Context:** Era 3 Phase 1 Track A — NATS Graduation

## New Subjects

The following subject conventions are added to the ADR-015 propagation list
for channel bridge traffic:

| Subject Pattern | Direction | Description |
|---|---|---|
| `dojo.channel.message.{platform}` | Inbound | Normalized ChannelMessage from platform webhook/actor |
| `dojo.channel.reply.{platform}` | Outbound | Reply from workflow execution back to platform |
| `dojo.channel.>` | Wildcard | Matches all channel events (used by ChannelBridge) |

### Platform Values

Current: `slack`, `discord`, `telegram`, `email`, `sms`, `webchat`, `teams`, `whatsapp`

### JetStream Configuration

Each platform gets a durable consumer with the following settings:

- **Consumer name:** `channel-{platform}` (e.g., `channel-slack`)
- **Durable:** Yes (survives restarts)
- **Ack policy:** Explicit
- **Max age (hot retention):** 30 days
- **Stream:** `DOJO_EVENTS` (existing stream from ADR-015)

### CloudEvent Type Mapping

Events on channel subjects carry CloudEvents v1.0 envelopes:

```json
{
  "specversion": "1.0",
  "type": "dojo.channel.message.slack",
  "source": "channel/slack",
  "id": "<uuid>",
  "time": "<rfc3339>",
  "datacontenttype": "application/json",
  "data": { /* ChannelMessage JSON */ }
}
```

## Rationale

InProcessBus was the Phase 0 synchronous stand-in for the event bus.
Phase 1 Track A graduates all channel traffic to the NATS JetStream bus,
providing:

1. **Durability** — messages survive process restarts via JetStream.
2. **Replay** — 30-day retention enables late-subscriber replay.
3. **Observability** — NATS subjects are traceable in the OTEL pipeline.
4. **Decoupling** — adapters and bridge communicate via subjects, not function calls.

InProcessBus is retained in test files only, not in any production code path.
