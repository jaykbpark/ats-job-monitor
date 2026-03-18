ALTER TABLE notifications ADD COLUMN kind TEXT NOT NULL DEFAULT 'new_match';

CREATE INDEX IF NOT EXISTS idx_notifications_watch_target_created
  ON notifications(watch_target_id, created_at DESC);
