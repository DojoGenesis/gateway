// Dojo Telemetry API — Cloudflare Worker + D1
// Ingests session events, tool calls, costs, spans, and orchestration plans.

interface Env {
  DB: D1Database;
}

// ---------------------------------------------------------------------------
// Event type constants (from Gateway catalog)
// ---------------------------------------------------------------------------

const TOOL_INVOKED = "tool_invoked";
const TOOL_COMPLETED = "tool_completed";
const COMPLETE = "complete";
const TRACE_SPAN_START = "trace_span_start";
const TRACE_SPAN_END = "trace_span_end";
const ORCHESTRATION_PLAN_CREATED = "orchestration_plan_created";
const ORCHESTRATION_COMPLETE = "orchestration_complete";
const ORCHESTRATION_FAILED = "orchestration_failed";

// ---------------------------------------------------------------------------
// CORS helpers
// ---------------------------------------------------------------------------

const CORS_HEADERS: Record<string, string> = {
  "Access-Control-Allow-Origin": "*",
  "Access-Control-Allow-Methods": "GET, POST, OPTIONS",
  "Access-Control-Allow-Headers": "Content-Type",
};

function json(data: unknown, status = 200): Response {
  return new Response(JSON.stringify(data), {
    status,
    headers: { "Content-Type": "application/json", ...CORS_HEADERS },
  });
}

function error(message: string, status = 400): Response {
  return json({ error: message }, status);
}

// ---------------------------------------------------------------------------
// Route: POST /api/telemetry/ingest
// ---------------------------------------------------------------------------

interface TelemetryEvent {
  type: string;
  ts: number;
  data?: Record<string, unknown>;
}

interface IngestPayload {
  session_id: string;
  events: TelemetryEvent[];
}

async function handleIngest(request: Request, db: D1Database): Promise<Response> {
  let body: IngestPayload;
  try {
    body = await request.json() as IngestPayload;
  } catch {
    return error("Invalid JSON body");
  }

  const { session_id, events } = body;
  if (!session_id || !Array.isArray(events) || events.length === 0) {
    return error("session_id (string) and events (non-empty array) required");
  }

  const stmts: D1PreparedStatement[] = [];

  // Ensure session row exists
  stmts.push(
    db.prepare("INSERT OR IGNORE INTO sessions (id, started_at) VALUES (?, ?)")
      .bind(session_id, events[0].ts ?? Date.now())
  );

  for (const evt of events) {
    const { type, ts, data } = evt;

    // Always insert into the raw events table
    stmts.push(
      db.prepare("INSERT INTO events (session_id, type, timestamp, data) VALUES (?, ?, ?, ?)")
        .bind(session_id, type, ts, data ? JSON.stringify(data) : null)
    );

    // --- Tool calls ---
    if (type === TOOL_INVOKED && data) {
      stmts.push(
        db.prepare(
          "INSERT INTO tool_calls (session_id, tool_name, started_at, arguments) VALUES (?, ?, ?, ?)"
        ).bind(
          session_id,
          (data.tool_name as string) ?? "",
          ts,
          data.arguments ? JSON.stringify(data.arguments) : null,
        )
      );
    }

    if (type === TOOL_COMPLETED && data) {
      // D1/SQLite does not support ORDER BY / LIMIT in UPDATE statements,
      // so we use a subquery to target the most recent uncompleted row.
      stmts.push(
        db.prepare(
          `UPDATE tool_calls
             SET completed_at    = ?,
                 duration_ms     = ? - started_at,
                 success         = ?,
                 result_summary  = ?
           WHERE rowid = (
             SELECT rowid FROM tool_calls
             WHERE session_id = ? AND tool_name = ? AND completed_at IS NULL
             ORDER BY started_at DESC LIMIT 1
           )`
        ).bind(
          ts,
          ts,
          data.success !== undefined ? (data.success ? 1 : 0) : 1,
          (data.result_summary as string) ?? null,
          session_id,
          (data.tool_name as string) ?? "",
        )
      );
    }

    // --- Costs (from "complete" events carrying usage) ---
    if (type === COMPLETE && data) {
      const usage = data.usage as Record<string, unknown> | undefined;
      if (usage) {
        stmts.push(
          db.prepare(
            "INSERT INTO costs (session_id, provider, model, tokens_in, tokens_out, cost_usd, timestamp) VALUES (?, ?, ?, ?, ?, ?, ?)"
          ).bind(
            session_id,
            (data.provider as string) ?? "unknown",
            (data.model as string) ?? "unknown",
            (usage.tokens_in as number) ?? 0,
            (usage.tokens_out as number) ?? 0,
            (usage.cost_usd as number) ?? 0,
            ts,
          )
        );
      }
    }

    // --- Trace spans ---
    if (type === TRACE_SPAN_START && data) {
      stmts.push(
        db.prepare(
          `INSERT INTO spans (span_id, trace_id, parent_id, session_id, name, start_time, status, inputs, metadata)
           VALUES (?, ?, ?, ?, ?, ?, 'started', ?, ?)`
        ).bind(
          (data.span_id as string) ?? "",
          (data.trace_id as string) ?? "",
          (data.parent_id as string) ?? null,
          session_id,
          (data.name as string) ?? "",
          ts,
          data.inputs ? JSON.stringify(data.inputs) : null,
          data.metadata ? JSON.stringify(data.metadata) : null,
        )
      );
    }

    if (type === TRACE_SPAN_END && data) {
      stmts.push(
        db.prepare(
          `UPDATE spans
             SET end_time    = ?,
                 duration_ms = ? - start_time,
                 status      = ?,
                 outputs     = ?
           WHERE span_id = ?`
        ).bind(
          ts,
          ts,
          (data.status as string) ?? "completed",
          data.outputs ? JSON.stringify(data.outputs) : null,
          (data.span_id as string) ?? "",
        )
      );
    }

    // --- Orchestration plans ---
    if (type === ORCHESTRATION_PLAN_CREATED && data) {
      stmts.push(
        db.prepare(
          `INSERT INTO orchestration_plans (plan_id, task_id, session_id, node_count, estimated_cost, total_nodes, status, created_at)
           VALUES (?, ?, ?, ?, ?, ?, 'running', ?)`
        ).bind(
          (data.plan_id as string) ?? "",
          (data.task_id as string) ?? "",
          session_id,
          (data.node_count as number) ?? 0,
          (data.estimated_cost as number) ?? 0,
          (data.total_nodes as number) ?? 0,
          ts,
        )
      );
    }

    if ((type === ORCHESTRATION_COMPLETE || type === ORCHESTRATION_FAILED) && data) {
      const status = type === ORCHESTRATION_COMPLETE ? "completed" : "failed";
      stmts.push(
        db.prepare(
          `UPDATE orchestration_plans
             SET status        = ?,
                 success_nodes = ?,
                 failed_nodes  = ?,
                 duration_ms   = ? - created_at,
                 completed_at  = ?
           WHERE plan_id = ?`
        ).bind(
          status,
          (data.success_nodes as number) ?? 0,
          (data.failed_nodes as number) ?? 0,
          ts,
          ts,
          (data.plan_id as string) ?? "",
        )
      );
    }
  }

  // Execute all statements in a single batch
  await db.batch(stmts);

  return json({ ingested: events.length });
}

// ---------------------------------------------------------------------------
// Route: GET /api/telemetry/sessions
// ---------------------------------------------------------------------------

async function handleSessions(url: URL, db: D1Database): Promise<Response> {
  const limit = Math.min(parseInt(url.searchParams.get("limit") ?? "20", 10), 100);

  const result = await db.prepare(
    `SELECT
       s.id,
       s.started_at,
       s.ended_at,
       COALESCE(s.total_cost, 0) AS total_cost,
       COALESCE(s.total_tokens, 0) AS total_tokens,
       COALESCE(s.total_tool_calls, 0) AS total_tool_calls,
       COALESCE(s.total_errors, 0) AS total_errors,
       (SELECT COUNT(*) FROM events e WHERE e.session_id = s.id) AS event_count
     FROM sessions s
     ORDER BY s.started_at DESC
     LIMIT ?`
  ).bind(limit).all();

  return json({ sessions: result.results });
}

// ---------------------------------------------------------------------------
// Route: GET /api/telemetry/costs
// ---------------------------------------------------------------------------

function rangeToSeconds(range: string): number {
  switch (range) {
    case "1d":  return 86_400;
    case "30d": return 2_592_000;
    case "7d":
    default:    return 604_800;
  }
}

async function handleCosts(url: URL, db: D1Database): Promise<Response> {
  const range = url.searchParams.get("range") ?? "7d";
  const since = Math.floor(Date.now() / 1000) - rangeToSeconds(range);

  // Individual cost rows
  const costsResult = await db.prepare(
    `SELECT id, session_id, provider, model, tokens_in, tokens_out, cost_usd, timestamp
     FROM costs
     WHERE timestamp >= ?
     ORDER BY timestamp DESC`
  ).bind(since).all();

  // Daily trend
  const trendResult = await db.prepare(
    `SELECT
       DATE(timestamp, 'unixepoch') AS day,
       SUM(cost_usd) AS total_cost,
       SUM(tokens_in + tokens_out) AS total_tokens
     FROM costs
     WHERE timestamp >= ?
     GROUP BY day
     ORDER BY day ASC`
  ).bind(since).all();

  // Summary: total + by-provider
  const summaryResult = await db.prepare(
    `SELECT
       provider,
       SUM(cost_usd) AS total_cost,
       SUM(tokens_in) AS total_tokens_in,
       SUM(tokens_out) AS total_tokens_out,
       COUNT(*) AS count
     FROM costs
     WHERE timestamp >= ?
     GROUP BY provider`
  ).bind(since).all();

  const totalCost = (summaryResult.results ?? []).reduce(
    (sum: number, r: Record<string, unknown>) => sum + ((r.total_cost as number) ?? 0),
    0,
  );

  return json({
    costs: costsResult.results,
    trend: trendResult.results,
    summary: {
      total_cost: totalCost,
      by_provider: summaryResult.results,
    },
  });
}

// ---------------------------------------------------------------------------
// Route: GET /api/telemetry/tools
// ---------------------------------------------------------------------------

async function handleTools(url: URL, db: D1Database): Promise<Response> {
  const range = url.searchParams.get("range") ?? "7d";
  const since = Math.floor(Date.now() / 1000) - rangeToSeconds(range);

  const result = await db.prepare(
    `SELECT
       tool_name          AS name,
       COUNT(*)           AS calls,
       ROUND(AVG(CASE WHEN duration_ms IS NOT NULL THEN duration_ms END), 1) AS avg_latency_ms,
       ROUND(
         CAST(SUM(CASE WHEN success = 1 THEN 1 ELSE 0 END) AS REAL) /
         CAST(COUNT(*) AS REAL) * 100,
         1
       ) AS success_rate
     FROM tool_calls
     WHERE started_at >= ?
     GROUP BY tool_name
     ORDER BY calls DESC`
  ).bind(since).all();

  return json({ tools: result.results });
}

// ---------------------------------------------------------------------------
// Route: GET /api/telemetry/spans
// ---------------------------------------------------------------------------

async function handleSpans(url: URL, db: D1Database): Promise<Response> {
  const sessionId = url.searchParams.get("session_id");
  const limit = Math.min(parseInt(url.searchParams.get("limit") ?? "100", 10), 500);

  let result;
  if (sessionId) {
    result = await db.prepare(
      `SELECT span_id, trace_id, parent_id, session_id, name,
              start_time, end_time, duration_ms, status, inputs, outputs, metadata
       FROM spans WHERE session_id = ? ORDER BY start_time ASC LIMIT ?`
    ).bind(sessionId, limit).all();
  } else {
    result = await db.prepare(
      `SELECT span_id, trace_id, parent_id, session_id, name,
              start_time, end_time, duration_ms, status, inputs, outputs, metadata
       FROM spans ORDER BY start_time DESC LIMIT ?`
    ).bind(limit).all();
  }

  return json({ spans: result.results });
}

// ---------------------------------------------------------------------------
// Route: GET /api/telemetry/orchestration
// ---------------------------------------------------------------------------

async function handleOrchestration(url: URL, db: D1Database): Promise<Response> {
  const range = url.searchParams.get("range") ?? "7d";
  const since = Math.floor(Date.now() / 1000) - rangeToSeconds(range);
  const limit = Math.min(parseInt(url.searchParams.get("limit") ?? "50", 10), 200);

  const result = await db.prepare(
    `SELECT plan_id, task_id, session_id, node_count, estimated_cost,
            total_nodes, status, created_at, completed_at,
            success_nodes, failed_nodes, duration_ms
     FROM orchestration_plans
     WHERE created_at >= ?
     ORDER BY created_at DESC
     LIMIT ?`
  ).bind(since, limit).all();

  // Summary stats
  const statsResult = await db.prepare(
    `SELECT
       COUNT(*) AS total,
       SUM(CASE WHEN status = 'completed' THEN 1 ELSE 0 END) AS completed,
       SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END) AS failed,
       ROUND(AVG(CASE WHEN duration_ms IS NOT NULL THEN duration_ms END), 0) AS avg_duration_ms
     FROM orchestration_plans
     WHERE created_at >= ?`
  ).bind(since).first();

  return json({
    plans: result.results,
    summary: statsResult,
  });
}

// ---------------------------------------------------------------------------
// Router
// ---------------------------------------------------------------------------

export default {
  async fetch(request: Request, env: Env): Promise<Response> {
    // Handle CORS preflight
    if (request.method === "OPTIONS") {
      return new Response(null, { status: 204, headers: CORS_HEADERS });
    }

    const url = new URL(request.url);
    const path = url.pathname;

    try {
      // POST /api/telemetry/ingest
      if (path === "/api/telemetry/ingest" && request.method === "POST") {
        return handleIngest(request, env.DB);
      }

      // GET /api/telemetry/sessions
      if (path === "/api/telemetry/sessions" && request.method === "GET") {
        return handleSessions(url, env.DB);
      }

      // GET /api/telemetry/costs
      if (path === "/api/telemetry/costs" && request.method === "GET") {
        return handleCosts(url, env.DB);
      }

      // GET /api/telemetry/tools
      if (path === "/api/telemetry/tools" && request.method === "GET") {
        return handleTools(url, env.DB);
      }

      // GET /api/telemetry/spans
      if (path === "/api/telemetry/spans" && request.method === "GET") {
        return handleSpans(url, env.DB);
      }

      // GET /api/telemetry/orchestration
      if (path === "/api/telemetry/orchestration" && request.method === "GET") {
        return handleOrchestration(url, env.DB);
      }

      return error("Not found", 404);
    } catch (e) {
      const message = e instanceof Error ? e.message : "Internal server error";
      return error(message, 500);
    }
  },
} satisfies ExportedHandler<Env>;
