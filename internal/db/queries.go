package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/fizza/fizza/internal/model"
)

func CreateProject(ctx context.Context, q Querier, name, description string) (*model.Project, error) {
	if err := model.ValidateProject(name, description); err != nil {
		return nil, err
	}
	txer, ok := q.(Transactor)
	if !ok {
		return nil, errors.New("db: CreateProject requires *sql.DB or *sql.Tx")
	}
	tx, err := txer.BeginTxx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("db: begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	res, err := tx.ExecContext(ctx,
		`INSERT INTO projects (name, description) VALUES (?, ?)`,
		name, description,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, fmt.Errorf("%w: project %q already exists", ErrDuplicate, name)
		}
		return nil, fmt.Errorf("db: insert project: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("db: last id: %w", err)
	}

	if _, err := tx.ExecContext(ctx,
		`INSERT INTO boards (project_id, name, is_default) VALUES (?, 'main', 1)`,
		id,
	); err != nil {
		return nil, fmt.Errorf("db: insert default board: %w", err)
	}
	var boardID int64
	if err := tx.QueryRowContext(ctx,
		`SELECT id FROM boards WHERE project_id = ? AND name = 'main'`, id,
	).Scan(&boardID); err != nil {
		return nil, fmt.Errorf("db: get default board id: %w", err)
	}
	for i, colName := range DefaultSeedColumns {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO columns (board_id, name, position) VALUES (?, ?, ?)`,
			boardID, colName, i+1,
		); err != nil {
			return nil, fmt.Errorf("db: insert column %q: %w", colName, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("db: commit: %w", err)
	}
	p, err := GetProject(ctx, q, id)
	if err != nil {
		return nil, err
	}
	projectID := p.ID
	_ = RecordEvent(ctx, q, Event{
		ProjectID: &projectID,
		Kind:      "project_create",
		Payload:   p.Name,
	})
	return p, nil
}

const projectSelect = `
	SELECT p.id, p.name, p.description, p.created_at, p.updated_at,
	       (SELECT COUNT(*) FROM boards b WHERE b.project_id = p.id) AS board_count
	FROM projects p`

func GetProject(ctx context.Context, q Querier, id int64) (*model.Project, error) {
	var p model.Project
	err := q.GetContext(ctx, &p, projectSelect+` WHERE p.id = ?`, id)
	if err != nil {
		return nil, mapErrNotFound(err, "project")
	}
	return &p, nil
}

func GetProjectByName(ctx context.Context, q Querier, name string) (*model.Project, error) {
	var p model.Project
	err := q.GetContext(ctx, &p, projectSelect+` WHERE p.name = ?`, name)
	if err != nil {
		return nil, mapErrNotFound(err, "project")
	}
	return &p, nil
}

func ListProjects(ctx context.Context, q Querier) ([]*model.Project, error) {
	var out []*model.Project
	err := q.SelectContext(ctx, &out, projectSelect+` ORDER BY p.name`)
	if err != nil {
		return nil, fmt.Errorf("db: list projects: %w", err)
	}
	return out, nil
}

func UpdateProject(ctx context.Context, q Querier, id int64, name, description string) (*model.Project, error) {
	if err := model.ValidateProject(name, description); err != nil {
		return nil, err
	}
	res, err := q.ExecContext(ctx, `
		UPDATE projects
		SET name = ?, description = ?, updated_at = strftime('%Y-%m-%dT%H:%M:%fZ','now')
		WHERE id = ?`, name, description, id)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, fmt.Errorf("%w: project %q already exists", ErrDuplicate, name)
		}
		return nil, fmt.Errorf("db: update project: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return nil, err
	}
	if n == 0 {
		return nil, fmt.Errorf("%w: project %d", ErrNotFound, id)
	}
	return GetProject(ctx, q, id)
}

func DeleteProject(ctx context.Context, q Querier, id int64) error {
	var name string
	if err := q.QueryRowContext(ctx, `SELECT name FROM projects WHERE id = ?`, id).Scan(&name); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("%w: project %d", ErrNotFound, id)
		}
		return fmt.Errorf("db: get project: %w", err)
	}

	txer, ok := q.(Transactor)
	if !ok {
		return errors.New("db: DeleteProject requires a Transactor")
	}
	tx, err := txer.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("db: begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, `
		DELETE FROM tasks WHERE board_id IN (SELECT id FROM boards WHERE project_id = ?)`, id); err != nil {
		return fmt.Errorf("db: delete project tasks: %w", err)
	}
	res, err := tx.ExecContext(ctx, `DELETE FROM projects WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("db: delete project: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return fmt.Errorf("%w: project %d", ErrNotFound, id)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("db: commit delete project: %w", err)
	}

	_ = RecordEvent(ctx, q, Event{
		Kind:    "project_delete",
		Payload: fmt.Sprintf("%d:%s", id, name),
	})
	return nil
}

func mapErrNotFound(err error, kind string) error {
	if errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("%w: %s", ErrNotFound, kind)
	}
	return fmt.Errorf("db: scan %s: %w", kind, err)
}
