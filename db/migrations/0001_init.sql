PRAGMA foreign_keys = ON;

CREATE TABLE IF NOT EXISTS watch_targets (
  id INTEGER PRIMARY KEY,
  name TEXT NOT NULL,
  provider TEXT NOT NULL,
  board_key TEXT NOT NULL,
  source_url TEXT,
  filters_json TEXT NOT NULL DEFAULT '{}',
  status TEXT NOT NULL DEFAULT 'active',
  last_synced_at TEXT,
  created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE(provider, board_key)
);

CREATE TABLE IF NOT EXISTS jobs (
  id INTEGER PRIMARY KEY,
  watch_target_id INTEGER NOT NULL,
  external_job_id TEXT NOT NULL,
  title TEXT NOT NULL,
  location TEXT,
  department TEXT,
  team TEXT,
  employment_type TEXT,
  job_url TEXT NOT NULL,
  metadata_json TEXT NOT NULL DEFAULT '{}',
  raw_json TEXT NOT NULL,
  first_seen_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
  last_seen_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
  matched_at TEXT,
  is_active INTEGER NOT NULL DEFAULT 1,
  FOREIGN KEY (watch_target_id) REFERENCES watch_targets(id) ON DELETE CASCADE,
  UNIQUE(watch_target_id, external_job_id)
);

CREATE TABLE IF NOT EXISTS sync_runs (
  id INTEGER PRIMARY KEY,
  watch_target_id INTEGER NOT NULL,
  status TEXT NOT NULL,
  started_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
  finished_at TEXT,
  fetched_jobs_count INTEGER NOT NULL DEFAULT 0,
  matched_jobs_count INTEGER NOT NULL DEFAULT 0,
  new_jobs_count INTEGER NOT NULL DEFAULT 0,
  error_message TEXT,
  FOREIGN KEY (watch_target_id) REFERENCES watch_targets(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS notifications (
  id INTEGER PRIMARY KEY,
  watch_target_id INTEGER NOT NULL,
  job_id INTEGER NOT NULL,
  channel TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'pending',
  sent_at TEXT,
  error_message TEXT,
  created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (watch_target_id) REFERENCES watch_targets(id) ON DELETE CASCADE,
  FOREIGN KEY (job_id) REFERENCES jobs(id) ON DELETE CASCADE,
  UNIQUE(job_id, channel)
);

CREATE INDEX IF NOT EXISTS idx_watch_targets_status
  ON watch_targets(status);

CREATE INDEX IF NOT EXISTS idx_jobs_watch_target_active
  ON jobs(watch_target_id, is_active);

CREATE INDEX IF NOT EXISTS idx_sync_runs_watch_target_started
  ON sync_runs(watch_target_id, started_at DESC);

CREATE INDEX IF NOT EXISTS idx_notifications_status
  ON notifications(status);
