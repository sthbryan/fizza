package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/fizza/fizza/internal/dbutil"
	"github.com/fizza/fizza/internal/model"
)

var DefaultSeedColumns = []string{"todo", "in_progress", "in_review", "done"}

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
	b, gerr := GetBoard(ctx, q, id)
	if gerr != nil {
		return nil, gerr
	}
	boardID := b.ID
	projID := b.ProjectID
	_ = RecordEvent(ctx, q, Event{
		ProjectID: &projID,
		BoardID:   &boardID,
		Kind:      "board_create",
		Payload:   b.Name,
	})
	return b, nil
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

func GetBoardByName(ctx context.Context, q Querier, projectID int64, name string) (*model.Board, error) {
	var b model.Board
	err := q.GetContext(ctx, &b,
		`SELECT id, project_id, name, is_default, created_at FROM boards
		 WHERE project_id = ? AND name = ?`, projectID, name)
	if err != nil {
		return nil, mapErrNotFound(err, fmt.Sprintf("board %q in project %d", name, projectID))
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
	var name string
	var projectID int64
	if err := q.QueryRowContext(ctx,
		`SELECT name, project_id FROM boards WHERE id = ?`, id,
	).Scan(&name, &projectID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("%w: board %d", ErrNotFound, id)
		}
		return fmt.Errorf("db: get board: %w", err)
	}

	txer, ok := q.(Transactor)
	if !ok {
		return errors.New("db: DeleteBoard requires a Transactor")
	}
	tx, err := txer.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("db: begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, `DELETE FROM tasks WHERE board_id = ?`, id); err != nil {
		return fmt.Errorf("db: delete board tasks: %w", err)
	}
	res, err := tx.ExecContext(ctx, `DELETE FROM boards WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("db: delete board: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("%w: board %d", ErrNotFound, id)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("db: commit delete board: %w", err)
	}

	_ = RecordEvent(ctx, q, Event{
		Kind:    "board_delete",
		Payload: fmt.Sprintf("%d:%s", id, name),
	})
	return nil
}

func GetColumnByName(ctx context.Context, q Querier, boardID int64, name string) (*model.Column, error) {
	var c model.Column
	err := q.GetContext(ctx, &c,
		`SELECT id, board_id, name, position, COALESCE(color, '') AS color, wip_limit FROM columns
		 WHERE board_id = ? AND name = ?`, boardID, name)
	if err != nil {
		return nil, mapErrNotFound(err, fmt.Sprintf("column %q in board %d", name, boardID))
	}
	return &c, nil
}

func ListColumns(ctx context.Context, q Querier, boardID int64) ([]*model.Column, error) {
	var out []*model.Column
	err := q.SelectContext(ctx, &out,
		`SELECT id, board_id, name, position, COALESCE(color, '') AS color, wip_limit FROM columns
		 WHERE board_id = ? ORDER BY position`, boardID)
	if err != nil {
		return nil, fmt.Errorf("db: list columns: %w", err)
	}
	return out, nil
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

func CreateColumn(ctx context.Context, q Querier, boardID int64, name string) (*model.Column, error) {
	if err := model.ValidateColumn(name); err != nil {
		return nil, err
	}
	var boardExists int64
	if err := q.QueryRowContext(ctx, `SELECT id FROM boards WHERE id = ?`, boardID).Scan(&boardExists); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%w: board %d", ErrNotFound, boardID)
		}
		return nil, fmt.Errorf("db: check board: %w", err)
	}

	var maxPos sql.NullInt64
	if err := q.QueryRowContext(ctx,
		`SELECT MAX(position) FROM columns WHERE board_id = ?`, boardID,
	).Scan(&maxPos); err != nil {
		return nil, fmt.Errorf("db: max column position: %w", err)
	}
	pos := 1
	if maxPos.Valid {
		pos = int(maxPos.Int64) + 1
	}

	res, err := q.ExecContext(ctx,
		`INSERT INTO columns (board_id, name, position) VALUES (?, ?, ?)`,
		boardID, name, pos,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, fmt.Errorf("%w: column %q already exists on board %d", ErrDuplicate, name, boardID)
		}
		return nil, fmt.Errorf("db: insert column: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("db: last id: %w", err)
	}
	return GetColumn(ctx, q, id)
}

func GetColumn(ctx context.Context, q Querier, id int64) (*model.Column, error) {
	var c model.Column
	err := q.GetContext(ctx, &c,
		`SELECT id, board_id, name, position, COALESCE(color, '') AS color, wip_limit FROM columns WHERE id = ?`, id)
	if err != nil {
		return nil, mapErrNotFound(err, fmt.Sprintf("column %d", id))
	}
	return &c, nil
}

func DeleteColumn(ctx context.Context, q Querier, boardID int64, name string, force bool) error {
	col, err := GetColumnByName(ctx, q, boardID, name)
	if err != nil {
		return err
	}

	var colCount int
	if err := q.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM columns WHERE board_id = ?`, boardID,
	).Scan(&colCount); err != nil {
		return fmt.Errorf("db: count columns: %w", err)
	}
	if colCount <= 1 {
		return fmt.Errorf("%w: board %d", ErrLastColumn, boardID)
	}

	var taskCount int
	if err := q.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM tasks WHERE column_id = ?`, col.ID,
	).Scan(&taskCount); err != nil {
		return fmt.Errorf("db: count column tasks: %w", err)
	}
	if taskCount > 0 && !force {
		return fmt.Errorf("%w: column %q has %d task(s); move them or pass force=true",
			ErrColumnNotEmpty, name, taskCount)
	}

	txer, ok := q.(Transactor)
	if !ok {
		return errors.New("db: DeleteColumn requires a Transactor")
	}
	tx, err := txer.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("db: begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if taskCount > 0 {
		if _, err := tx.ExecContext(ctx, `DELETE FROM tasks WHERE column_id = ?`, col.ID); err != nil {
			return fmt.Errorf("db: delete column tasks: %w", err)
		}
	}
	res, err := tx.ExecContext(ctx, `DELETE FROM columns WHERE id = ?`, col.ID)
	if err != nil {
		return fmt.Errorf("db: delete column: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("%w: column %q", ErrNotFound, name)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("db: commit delete column: %w", err)
	}

	projectID := boardProjectID(ctx, q, boardID)
	boardIDCopy := boardID
	_ = RecordEvent(ctx, q, Event{
		ProjectID: projectID,
		BoardID:   &boardIDCopy,
		Kind:      "column_delete",
		Payload:   fmt.Sprintf("%d:%s", col.ID, name),
	})
	return nil
}
