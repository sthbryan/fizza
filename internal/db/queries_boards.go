package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/fizza/fizza/internal/dbutil"
	"github.com/fizza/fizza/internal/model"
)

var DefaultSeedColumns = []string{"todo", "in_progress", "done"}

func CreateBoard(ctx context.Context, q Querier, projectID int64, name string) (*model.Board, error) {
	return CreateBoardWithColumns(ctx, q, projectID, name, DefaultSeedColumns)
}

func CreateBoardWithColumns(ctx context.Context, q Querier, projectID int64, name string, columns []string) (*model.Board, error) {
	if err := model.ValidateBoard(name); err != nil {
		return nil, err
	}
	if len(columns) == 0 {
		columns = DefaultSeedColumns
	}
	for _, c := range columns {
		if err := model.ValidateColumn(c); err != nil {
			return nil, fmt.Errorf("seed column %q: %w", c, err)
		}
	}

	txer, ok := q.(Transactor)
	if !ok {
		return nil, errors.New("db: CreateBoardWithColumns requires *sql.DB or *sql.Tx")
	}
	tx, err := txer.BeginTxx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("db: begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var existing int
	if err := tx.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM boards WHERE project_id = ?`, projectID).Scan(&existing); err != nil {
		return nil, fmt.Errorf("db: count boards: %w", err)
	}
	isDefault := existing == 0

	res, err := tx.ExecContext(ctx,
		`INSERT INTO boards (project_id, name, is_default) VALUES (?, ?, ?)`,
		projectID, name, dbutil.BoolToInt(isDefault),
	)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, fmt.Errorf("%w: board %q already exists in project %d", ErrDuplicate, name, projectID)
		}
		return nil, fmt.Errorf("db: insert board: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("db: last id: %w", err)
	}

	for i, colName := range columns {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO columns (board_id, name, position) VALUES (?, ?, ?)`,
			id, colName, i+1,
		); err != nil {
			if isUniqueViolation(err) {
				return nil, fmt.Errorf("%w: column %q already exists", ErrDuplicate, colName)
			}
			return nil, fmt.Errorf("db: insert column %q: %w", colName, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("db: commit: %w", err)
	}
	return GetBoard(ctx, q, id)
}

func GetBoard(ctx context.Context, q Querier, id int64) (*model.Board, error) {
	var b model.Board
	err := q.GetContext(ctx, &b,
		`SELECT id, project_id, name, is_default, created_at FROM boards WHERE id = ?`, id)
	if err != nil {
		return nil, mapErrNotFound(err, fmt.Sprintf("board %d", id))
	}
	return &b, nil
}

func ListBoards(ctx context.Context, q Querier, projectID int64) ([]*model.Board, error) {
	var out []*model.Board
	err := q.SelectContext(ctx, &out,
		`SELECT id, project_id, name, is_default, created_at
		 FROM boards WHERE project_id = ? ORDER BY name`, projectID)
	if err != nil {
		return nil, fmt.Errorf("db: list boards: %w", err)
	}
	return out, nil
}

func DeleteBoard(ctx context.Context, q Querier, id int64) error {
	res, err := q.ExecContext(ctx, `DELETE FROM boards WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("db: delete board: %w (move or delete tasks first)", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("%w: board %d", ErrNotFound, id)
	}
	return nil
}

func GetColumnByName(ctx context.Context, q Querier, boardID int64, name string) (*model.Column, error) {
	var (
		c    model.Column
		pos  int
		c2   sql.NullString
		wip  sql.NullInt64
	)
	err := q.QueryRowContext(ctx,
		`SELECT id, board_id, name, position, color, wip_limit FROM columns
		 WHERE board_id = ? AND name = ?`, boardID, name,
	).Scan(&c.ID, &c.BoardID, &c.Name, &pos, &c2, &wip)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%w: column %q in board %d", ErrNotFound, name, boardID)
		}
		return nil, fmt.Errorf("db: get column: %w", err)
	}
	c.Position = pos
	if c2.Valid {
		c.Color = c2.String
	}
	if wip.Valid {
		v := int(wip.Int64)
		c.WIPLimit = &v
	}
	return &c, nil
}

func ListColumns(ctx context.Context, q Querier, boardID int64) ([]*model.Column, error) {
	rows, err := q.QueryContext(ctx,
		`SELECT id, board_id, name, position, color, wip_limit FROM columns
		 WHERE board_id = ? ORDER BY position`, boardID)
	if err != nil {
		return nil, fmt.Errorf("db: list columns: %w", err)
	}
	defer rows.Close()
	var out []*model.Column
	for rows.Next() {
		var (
			c    model.Column
			pos  int
			colr sql.NullString
			wip  sql.NullInt64
		)
		if err := rows.Scan(&c.ID, &c.BoardID, &c.Name, &pos, &colr, &wip); err != nil {
			return nil, err
		}
		c.Position = pos
		if colr.Valid {
			c.Color = colr.String
		}
		if wip.Valid {
			v := int(wip.Int64)
			c.WIPLimit = &v
		}
		out = append(out, &c)
	}
	return out, rows.Err()
}

func UpdateColumnWIPLimit(ctx context.Context, q Querier, columnID int64, limit *int) error {
	var wipParam any
	if limit != nil {
		wipParam = *limit
	}
	var existing int64
	err := q.QueryRowContext(ctx, `SELECT id FROM columns WHERE id = ?`, columnID).Scan(&existing)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("%w: column %d", ErrNotFound, columnID)
		}
		return fmt.Errorf("db: check column: %w", err)
	}
	if _, err := q.ExecContext(ctx, `UPDATE columns SET wip_limit = ? WHERE id = ?`, wipParam, columnID); err != nil {
		return fmt.Errorf("db: update wip limit: %w", err)
	}
	return nil
}