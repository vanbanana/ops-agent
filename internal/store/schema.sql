-- ops-agent schema v1
-- 5 tables as per design.md §3.2

PRAGMA journal_mode=WAL;
PRAGMA foreign_keys=ON;

CREATE TABLE IF NOT EXISTS sessions (
    id         TEXT PRIMARY KEY,
    title      TEXT NOT NULL DEFAULT '新对话',
    user_id    TEXT NOT NULL DEFAULT 'admin',
    mode       TEXT NOT NULL DEFAULT 'auto' CHECK(mode IN ('auto','single','multi')),
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
);

CREATE TABLE IF NOT EXISTS messages (
    id           TEXT PRIMARY KEY,
    session_id   TEXT NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    role         TEXT NOT NULL CHECK(role IN ('user','assistant','tool','system')),
    content      TEXT NOT NULL DEFAULT '',
    tool_calls   TEXT,
    tool_call_id TEXT,
    tool_name    TEXT,
    token_count  INTEGER,
    created_at   TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
);
CREATE INDEX IF NOT EXISTS idx_messages_session ON messages(session_id, created_at);

CREATE TABLE IF NOT EXISTS audit_logs (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    trace_id     TEXT NOT NULL,
    session_id   TEXT NOT NULL,
    round_number INTEGER NOT NULL DEFAULT 1,
    stage        TEXT NOT NULL CHECK(stage IN ('SENSE','ANALYZE','PLAN','EXECUTE','OUTPUT')),
    role         TEXT,
    content      TEXT NOT NULL DEFAULT '',
    metadata     TEXT,
    triggered_by TEXT NOT NULL DEFAULT 'llm' CHECK(triggered_by IN ('llm','ui_click','terminal')),
    status       TEXT NOT NULL DEFAULT 'ok' CHECK(status IN ('ok','warning','blocked','error')),
    duration_ms  INTEGER NOT NULL DEFAULT 0,
    created_at   TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
);
CREATE INDEX IF NOT EXISTS idx_audit_trace ON audit_logs(trace_id, created_at);
CREATE INDEX IF NOT EXISTS idx_audit_session ON audit_logs(session_id, created_at DESC);

CREATE TABLE IF NOT EXISTS configs (
    key         TEXT PRIMARY KEY,
    value       TEXT NOT NULL DEFAULT '',
    description TEXT,
    updated_at  TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
    updated_by  TEXT
);

CREATE TABLE IF NOT EXISTS mcp_servers (
    id         TEXT PRIMARY KEY,
    name       TEXT NOT NULL,
    transport  TEXT NOT NULL CHECK(transport IN ('stdio','http')),
    command    TEXT,
    args       TEXT NOT NULL DEFAULT '[]',
    env        TEXT NOT NULL DEFAULT '{}',
    url        TEXT,
    is_active  INTEGER NOT NULL DEFAULT 1,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
);

-- Schema version tracking
INSERT OR IGNORE INTO configs (key, value, description) VALUES ('schema_version', '1', 'Current DB schema version');
