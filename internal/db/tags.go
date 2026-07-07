package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/fizza/fizza/internal/model"
)

func CreateTag(ctx context.Context, q Querier, projectID int64, name string) (*model.Tag, error) {
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

func GetTag(ctx context.Context, q Querier, id int64) (*model.Tag, error) {
	var t model.Tag
	err := q.GetContext(ctx, &t,
		`SELECT id, project_id, name, created_at FROM tags WHERE id = ?`, id)
	if err != nil {
		return nil, mapErrNotFound(err, fmt.Sprintf("tag %d", id))
	}
	return &t, nil
}

func ListTags(ctx context.Context, q Querier, projectID int64) ([]*model.Tag, error) {
	var out []*model.Tag
	err := q.SelectContext(ctx, &out,
		`SELECT id, project_id, name, created_at FROM tags
		 WHERE project_id = ? ORDER BY name`, projectID)
	if err != nil {
		return nil, fmt.Errorf("db: list tags: %w", err)
	}
	return out, nil
}

func DeleteTag(ctx context.Context, q Querier, tagID int64) error {
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

func AddTagToTask(ctx context.Context, q Querier, taskID, tagID int64) error {
	_, err := q.ExecContext(ctx,
		`INSERT OR IGNORE INTO task_tags (task_id, tag_id) VALUES (?, ?)`,
		taskID, tagID)
	if err != nil {
		return fmt.Errorf("db: add tag to task: %w", err)
	}
	return nil
}

func RemoveTagFromTask(ctx context.Context, q Querier, taskID, tagID int64) error {
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

func ListTagsForTask(ctx context.Context, q Querier, taskID int64) ([]*model.Tag, error) {
	var out []*model.Tag
	err := q.SelectContext(ctx, &out, `
		SELECT t.id, t.project_id, t.name, t.created_at
		FROM tags t
		JOIN task_tags tt ON tt.tag_id = t.id
		WHERE tt.task_id = ?
		ORDER BY t.name`, taskID)
	if err != nil {
		return nil, fmt.Errorf("db: list tags for task: %w", err)
	}
	return out, nil
}

func ListTaskIDsForTag(ctx context.Context, q Querier, tagID int64) ([]int64, error) {
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

type TaskTagChanges struct {
	Add    []string
	Remove []string
}

func ApplyTaskTagChanges(ctx context.Context, q Querier, taskID, projectID int64, changes TaskTagChanges) error {
	if len(changes.Add) == 0 && len(changes.Remove) == 0 {
		return nil
	}
	names := append([]string{}, changes.Add...)
	names = append(names, changes.Remove...)
	tags, err := resolveTagsByName(ctx, q, projectID, names)
	if err != nil {
		return err
	}
	if txer, ok := q.(Transactor); ok {
		tx, err := txer.BeginTxx(ctx, nil)
		if err != nil {
			return fmt.Errorf("db: begin tx: %w", err)
		}
		committed := false
		defer func() {
			if !committed {
				_ = tx.Rollback()
			}
		}()
		for _, name := range changes.Add {
			tag := tags[name]
			if tag == nil {
				created, err := CreateTag(ctx, tx, projectID, name)
				if err != nil {
					return err
				}
				tag = created
			}
			if err := AddTagToTask(ctx, tx, taskID, tag.ID); err != nil {
				return err
			}
		}
		for _, name := range changes.Remove {
			tag := tags[name]
			if tag == nil {
				continue
			}
			if err := removeTagFromTaskIgnoreMissing(ctx, tx, taskID, tag.ID); err != nil {
				return err
			}
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("db: commit: %w", err)
		}
		committed = true
		return nil
	}
	for _, name := range changes.Add {
		tag := tags[name]
		if tag == nil {
			created, err := CreateTag(ctx, q, projectID, name)
			if err != nil {
				return err
			}
			tag = created
		}
		if err := AddTagToTask(ctx, q, taskID, tag.ID); err != nil {
			return err
		}
	}
	for _, name := range changes.Remove {
		tag := tags[name]
		if tag == nil {
			continue
		}
		if err := removeTagFromTaskIgnoreMissing(ctx, q, taskID, tag.ID); err != nil {
			return err
		}
	}
	return nil
}

func resolveTagsByName(ctx context.Context, q Querier, projectID int64, names []string) (map[string]*model.Tag, error) {
	out := make(map[string]*model.Tag, len(names))
	for _, n := range names {
		if n == "" {
			continue
		}
		var t model.Tag
		err := q.GetContext(ctx, &t,
			`SELECT id, project_id, name, created_at FROM tags WHERE project_id = ? AND name = ?`,
			projectID, n)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("db: resolve tag %q: %w", n, err)
		}
		if err == nil {
			out[n] = &t
		}
	}
	return out, nil
}

func removeTagFromTaskIgnoreMissing(ctx context.Context, q Querier, taskID, tagID int64) error {
	_, err := q.ExecContext(ctx,
		`DELETE FROM task_tags WHERE task_id = ? AND tag_id = ?`,
		taskID, tagID)
	if err != nil {
		return fmt.Errorf("db: remove tag from task: %w", err)
	}
	return nil
}
