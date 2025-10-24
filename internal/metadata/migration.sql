-- migrations for metadata sqlite
CREATE TABLE IF NOT EXISTS backups (
  id TEXT PRIMARY KEY,
  name TEXT,
  type TEXT,
  storage_path TEXT,
  started_at INTEGER,
  finished_at INTEGER,
  size INTEGER
);

CREATE INDEX IF NOT EXISTS idx_backups_finished_at ON backups(finished_at);

CREATE TABLE IF NOT EXISTS schedules (
  id TEXT PRIMARY KEY,
  db_type TEXT,
  source TEXT,
  cron_expr TEXT,
  retention_days INTEGER,
  created_at INTEGER,
  last_run INTEGER
);
