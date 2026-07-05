CREATE TABLE events (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    project_id INTEGER REFERENCES projects(id) ON DELETE CASCADE,
    board_id   INTEGER REFERENCES boards(id)   ON DELETE CASCADE,
    task_id    INTEGER REFERENCES tasks(id)    ON DELETE CASCADE,
    kind       TEXT    NOT NULL,
    payload    TEXT    NOT NULL DEFAULT '',
    created_at TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
    CHECK(kind IN ('project_create','project_delete',
                   'board_create','board_delete',
                   'task_create','task_update','task_move','task_delete'))
);

CREATE INDEX idx_events_task    ON events(task_id, created_at DESC);
CREATE INDEX idx_events_project ON events(project_id, created_at DESC);