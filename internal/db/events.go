package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/fizza/fizza/internal/dbutil"
	"github.com/fizza/fizza/internal/model"
)

type Event struct {
	ID        int64     `json:"id"`
	ProjectID *int64    `json:"project_id,omitempty"`
	BoardID   *int64    `json:"board_id,omitempty"`
	TaskID    *int64    `json:"task_id,omitempty"`
	Kind      string    `json:"kind"`
	Payload   string    `json:"payload,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

var allowedEventKinds = map[string]bool{
	"project_create": true,
	"project_delete": true,
	"board_create":   true,
	"board_delete":   true,
	"task_create":    true,
	"task_update":    true,
	"task_move":      true,
	"task_delete":    true,
}

func RecordEvent(ctx context.Context, q Querier, ev Event) error {
	if !allowedEventKinds[ev.Kind] {
		return fmt.Errorf("db: invalid event kind %q", ev.Kind)
	}
	_, err := q.ExecContext(ctx, `
		INSERT INTO events (project_id, board_id, task_id, kind, payload)
		VALUES (?, ?, ?, ?, ?)`,
		nullableInt64(ev.ProjectID),
		nullableInt64(ev.BoardID),
		nullableInt64(ev.TaskID),
		ev.Kind,
		ev.Payload,
	)
	if err != nil {
		return fmt.Errorf("db: insert event: %w", err)
	}
	return nil
}

func nullableInt64(p *int64) any {
	if p == nil {
		return nil
	}
	return *p
}

func ListEvents(ctx context.Context, q Querier, taskID *int64, limit int) ([]*model.Event, error) {
	if limit <= 0 || limit > 500 {
		limit = 50
	}
	args := []any{}
	where := ""
	if taskID != nil {
		where = "WHERE task_id = ?"
		args = append(args, *taskID)
	}
	args = append(args, limit)
	rows, err := q.QueryContext(ctx,
		`SELECT id, project_id, board_id, task_id, kind, payload, created_at
		 FROM events `+where+` ORDER BY created_at DESC, id DESC LIMIT ?`, args...)
	if err != nil {
		return nil, fmt.Errorf("db: list events: %w", err)
	}
	defer rows.Close()
	var out []*model.Event
	for rows.Next() {
		var (
			ev       model.Event
			proj, b  sql.NullInt64
			taskCol  sql.NullInt64
			payload  sql.NullString
			created  string
		)
		if err := rows.Scan(&ev.ID, &proj, &b, &taskCol, &ev.Kind, &payload, &created); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return out, nil
			}
			return nil, err
		}
		if proj.Valid {
			v := proj.Int64
			ev.ProjectID = &v
		}
		if b.Valid {
			v := b.Int64
			ev.BoardID = &v
		}
		if taskCol.Valid {
			v := taskCol.Int64
			ev.TaskID = &v
		}
		if payload.Valid {
			ev.Payload = payload.String
		}
		t, err := parseTimeLocal(created)
		if err != nil {
			return nil, err
		}
		ev.CreatedAt = t
		out = append(out, &ev)
	}
	return out, rows.Err()
}

func parseTimeLocal(s string) (dbutil.Time, error) {
	for _, layout := range []string{
		"2006-01-02T15:04:05.000000Z07:00",
		"2006-01-02T15:04:05.000Z07:00",
		time.RFC3339Nano,
		time.RFC3339,
	} {
		if t, err := time.Parse(layout, s); err == nil {
			return dbutil.Time{Time: t.UTC()}, nil
		}
	}
	return dbutil.Time{}, fmt.Errorf("unparseable timestamp %q", s)
}