ALTER TABLE jobs ADD COLUMN is_match INTEGER NOT NULL DEFAULT 1;
ALTER TABLE jobs ADD COLUMN match_reasons_json TEXT NOT NULL DEFAULT '[]';
ALTER TABLE jobs ADD COLUMN hard_failures_json TEXT NOT NULL DEFAULT '[]';

CREATE INDEX IF NOT EXISTS idx_jobs_watch_target_match
  ON jobs(watch_target_id, is_match, is_active);
