-- Dojo Telemetry D1 Schema
-- Migration 0001: initial schema

CREATE TABLE IF NOT EXISTS sessions (
  id          TEXT    PRIMARY KEY,
  started_at  INTEGER NOT NULL,
  ended_at    INTEGER,
  total_cost  REAL,
  total_tokens INTEGER,
  total_tool_calls INTEGER,
  total_errors INTEGER
);

CREATE TABLE IF NOT EXISTS events (
  id          INTEGER PRIMARY KEY AUTOINCREMENT,
  session_id  TEXT    NOT NULL,
  type        TEXT    NOT NULL,
  timestamp   INTEGER NOT NULL,
  data        TEXT
);

CREATE TABLE IF NOT EXISTS tool_calls (
  id              INTEGER PRIMARY KEY AUTOINCREMENT,
  session_id      TEXT    NOT NULL,
  tool_name       TEXT    NOT NULL,
  started_at      INTEGER NOT NULL,
  arguments       TEXT,
  completed_at    INTEGER,
  duration_ms     INTEGER,
  success         INTEGER,
  result_summary  TEXT
);

CREATE TABLE IF NOT EXISTS costs (
  id          INTEGER PRIMARY KEY AUTOINCREMENT,
  session_id  TEXT    NOT NULL,
  provider    TEXT,
  model       TEXT,
  tokens_in   INTEGER,
  tokens_out  INTEGER,
  cost_usd    REAL,
  timestamp   INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS spans (
  span_id     TEXT    PRIMARY KEY,
  trace_id    TEXT,
  parent_id   TEXT,
  session_id  TEXT    NOT NULL,
  name        TEXT,
  start_time  INTEGER,
  end_time    INTEGER,
  duration_ms INTEGER,
  status      TEXT,
  inputs      TEXT,
  outputs     TEXT,
  metadata    TEXT
);

CREATE TABLE IF NOT EXISTS orchestration_plans (
  plan_id        TEXT    PRIMARY KEY,
  task_id        TEXT,
  session_id     TEXT    NOT NULL,
  node_count     INTEGER,
  estimated_cost REAL,
  total_nodes    INTEGER,
  status         TEXT    DEFAULT 'running',
  created_at     INTEGER NOT NULL,
  completed_at   INTEGER,
  success_nodes  INTEGER,
  failed_nodes   INTEGER,
  duration_ms    INTEGER
);

CREATE INDEX IF NOT EXISTS idx_events_session    ON events(session_id);
CREATE INDEX IF NOT EXISTS idx_tool_calls_session ON tool_calls(session_id);
CREATE INDEX IF NOT EXISTS idx_costs_session      ON costs(session_id);
CREATE INDEX IF NOT EXISTS idx_spans_session      ON spans(session_id);
CREATE INDEX IF NOT EXISTS idx_orch_session       ON orchestration_plans(session_id);
