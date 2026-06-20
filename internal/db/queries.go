package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/fizza/fizza/internal/dbutil"
	"github.com/fizza/fizza/internal/model"
)

type querier interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

type transactor interface {
	BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error)
}

type rowScanner interface {
	Scan(dest ...any) error
}

func CreateProject(ctx context.Context, q querier, name, description string) (*model.Project, error) {
	if err := model.ValidateProject(name, description); err != nil {
		return nil, err
	}
	res, err := q.ExecContext(ctx,
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
	return GetProject(ctx, q, id)
}

func GetProject(ctx context.Context, q querier, id int64) (*model.Project, error) {
	row := q.QueryRowContext(ctx,
		`SELECT id, name, description, created_at, updated_at FROM projects WHERE id = ?`, id)
	return scanProject(row)
}

func GetProjectByName(ctx context.Context, q querier, name string) (*model.Project, error) {
	row := q.QueryRowContext(ctx,
		`SELECT id, name, description, created_at, updated_at FROM projects WHERE name = ?`, name)
	return scanProject(row)
}

func ListProjects(ctx context.Context, q querier) ([]*model.Project, error) {
	rows, err := q.QueryContext(ctx,
		`SELECT id, name, description, created_at, updated_at FROM projects ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("db: list projects: %w", err)
	}
	defer rows.Close()
	var out []*model.Project
	for rows.Next() {
		p, err := scanProject(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func DeleteProject(ctx context.Context, q querier, id int64) error {
	res, err := q.ExecContext(ctx, `DELETE FROM projects WHERE id = ?`, id)
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
	return nil
}

func scanProject(s rowScanner) (*model.Project, error) {
	var (
		p     model.Project
		creAt string
		updAt string
	)
	if err := s.Scan(&p.ID, &p.Name, &p.Description, &creAt, &updAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%w: project", ErrNotFound)
		}
		return nil, fmt.Errorf("db: scan project: %w", err)
	}
	var err error
	if p.CreatedAt, err = dbutil.ParseTime(creAt); err != nil {
		return nil, fmt.Errorf("db: parse created_at: %w", err)
	}
	if p.UpdatedAt, err = dbutil.ParseTime(updAt); err != nil {
		return nil, fmt.Errorf("db: parse updated_at: %w", err)
	}
	return &p, nil
}