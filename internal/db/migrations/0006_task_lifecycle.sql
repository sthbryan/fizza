ALTER TABLE tasks ADD COLUMN completed_at TEXT;
ALTER TABLE tasks ADD COLUMN archived_at TEXT;

CREATE INDEX idx_tasks_board_archived ON tasks(board_id) WHERE archived_at IS NOT NULL;
CREATE INDEX idx_tasks_board_completed ON tasks(board_id) WHERE completed_at IS NOT NULL;
