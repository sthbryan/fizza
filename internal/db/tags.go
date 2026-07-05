package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/fizza/fizza/internal/dbutil"
	"github.com/fizza/fizza/internal/model"
)

func CreateTag(ctx context.Context, q querier, projectID int64, name string) (*model.Tag, error) {
	if err := model.ValidateTag(name); err != nil {
		return nil, err
	}
	res, err := q.ExecContext(ctx,
		`INSERT INTO tags (project_id, name) VALUES (?, ?)`,
		projectID, name,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, fmt.Errorf("%w: tag %q in project %d", ErrDuplicate, name, projectID)
		}
		return nil, fmt.Errorf("db: insert tag: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("db: last id: %w", err)
	}
	return GetTag(ctx, q, id)
}

func GetTag(ctx context.Context, q querier, id int64) (*model.Tag, error) {
	row := q.QueryRowContext(ctx,
		`SELECT id, project_id, name, created_at FROM tags WHERE id = ?`, id)
	return scanTag(row)
}

func ListTags(ctx context.Context, q querier, projectID int64) ([]*model.Tag, error) {
	rows, err := q.QueryContext(ctx,
		`SELECT id, project_id, name, created_at FROM tags
		 WHERE project_id = ? ORDER BY name`, projectID)
	if err != nil {
		return nil, fmt.Errorf("db: list tags: %w", err)
	}
	defer rows.Close()
	var out []*model.Tag
	for rows.Next() {
		t, err := scanTag(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func DeleteTag(ctx context.Context, q querier, tagID int64) error {
	res, err := q.ExecContext(ctx, `DELETE FROM tags WHERE id = ?`, tagID)
	if err != nil {
		return fmt.Errorf("db: delete tag: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return fmt.Errorf("%w: tag %d", ErrNotFound, tagID)
	}
	return nil
}

func AddTagToTask(ctx context.Context, q querier, taskID, tagID int64) error {
	_, err := q.ExecContext(ctx,
		`INSERT OR IGNORE INTO task_tags (task_id, tag_id) VALUES (?, ?)`,
		taskID, tagID)
	if err != nil {
		return fmt.Errorf("db: add tag to task: %w", err)
	}
	return nil
}

func RemoveTagFromTask(ctx context.Context, q querier, taskID, tagID int64) error {
	res, err := q.ExecContext(ctx,
		`DELETE FROM task_tags WHERE task_id = ? AND tag_id = ?`,
		taskID, tagID)
	if err != nil {
		return fmt.Errorf("db: remove tag from task: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("%w: task %d tag %d", ErrNotFound, taskID, tagID)
	}
	return nil
}

func ListTagsForTask(ctx context.Context, q querier, taskID int64) ([]*model.Tag, error) {
	rows, err := q.QueryContext(ctx, `
		SELECT t.id, t.project_id, t.name, t.created_at
		FROM tags t
		JOIN task_tags tt ON tt.tag_id = t.id
		WHERE tt.task_id = ?
		ORDER BY t.name`, taskID)
	if err != nil {
		return nil, fmt.Errorf("db: list tags for task: %w", err)
	}
	defer rows.Close()
	var out []*model.Tag
	for rows.Next() {
		t, err := scanTag(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func ListTaskIDsForTag(ctx context.Context, q querier, tagID int64) ([]int64, error) {
	rows, err := q.QueryContext(ctx,
		`SELECT task_id FROM task_tags WHERE tag_id = ? ORDER BY task_id`, tagID)
	if err != nil {
		return nil, fmt.Errorf("db: list task ids for tag: %w", err)
	}
	defer rows.Close()
	var out []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

func scanTag(s rowScanner) (*model.Tag, error) {
	var (
		t     model.Tag
		creAt string
	)
	if err := s.Scan(&t.ID, &t.ProjectID, &t.Name, &creAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%w: tag", ErrNotFound)
		}
		return nil, fmt.Errorf("db: scan tag: %w", err)
	}
	parsed, err := dbutil.ParseTime(creAt)
	if err != nil {
		return nil, fmt.Errorf("db: parse created_at: %w", err)
	}
	t.CreatedAt = parsed
	return &t, nil
}