package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/fizza/fizza/internal/dbutil"
	"github.com/fizza/fizza/internal/model"
)

const taskSelect = `
	SELECT t.id, t.board_id, t.column_id, c.name AS status,
	       t.parent_id, t.title, t.description, t.priority, t.position,
	       t.due_date, t.created_at, t.updated_at
	FROM tasks t
	JOIN columns c ON c.id = t.column_id`

type TaskPatch struct {
	Title         *string
	Description   *string
	Priority      *string
	DueDate       *time.Time
	ClearDueDate  bool
	ParentID      *int64
	ClearParentID bool
}

func CreateTask(ctx context.Context, q querier, t *model.Task) error {
	if t.ColumnID == 0 {
		return errors.New("db: task.ColumnID required")
	}
	if strings.TrimSpace(t.Priority) == "" {
		t.Priority = model.DefaultPriority
	}
	if err := t.Validate(); err != nil {
		return err
	}

	var pos float64
	var err error
	if t.Position == 0 {
		pos, err = computeNextPosition(ctx, q, t.ColumnID, nil)
	} else {
		pos = t.Position
	}
	if errors.Is(err, errGapTooSmall) {
		if _, rerr := RebalanceColumn(ctx, q, t.ColumnID); rerr != nil {
			return fmt.Errorf("db: rebalance before insert: %w", rerr)
		}
		pos, err = computeNextPosition(ctx, q, t.ColumnID, nil)
	}
	if err != nil {
		return fmt.Errorf("db: compute position: %w", err)
	}

	var dueParam any
	if t.DueDate != nil {
		dueParam = dbutil.FormatDueDate(*t.DueDate)
	}

	res, err := q.ExecContext(ctx, `
		INSERT INTO tasks
		  (board_id, column_id, parent_id, title, description, priority, position, due_date)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		t.BoardID, t.ColumnID, dbutil.NullableInt(t.ParentID), t.Title, t.Description, t.Priority, pos, dueParam,
	)
	if err != nil {
		return fmt.Errorf("db: insert task: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return fmt.Errorf("db: last id: %w", err)
	}
	t.ID = id
	t.Position = pos

	ensureHealthyPosition(ctx, q, t.ColumnID)

	fresh, err := GetTask(ctx, q, id)
	if err != nil {
		return err
	}
	*t = *fresh
	return nil
}

func GetTask(ctx context.Context, q querier, id int64) (*model.Task, error) {
	row := q.QueryRowContext(ctx, taskSelect+` WHERE t.id = ?`, id)
	return scanTask(row)
}

func GetTaskByPrefix(ctx context.Context, q querier, prefix string) (*model.Task, error) {
	prefix = strings.TrimSpace(prefix)
	if prefix == "" {
		return nil, errors.New("db: empty prefix")
	}
	if !dbutil.IsDigits(prefix) {
		return nil, fmt.Errorf("db: task prefix must be numeric: %q", prefix)
	}
	rows, err := q.QueryContext(ctx,
		taskSelect+` WHERE CAST(t.id AS TEXT) LIKE ? ORDER BY t.id LIMIT 2`,
		prefix+"%",
	)
	if err != nil {
		return nil, fmt.Errorf("db: prefix lookup: %w", err)
	}
	defer rows.Close()

	var matches []*model.Task
	for rows.Next() {
		mt, err := scanTask(rows)
		if err != nil {
			return nil, err
		}
		matches = append(matches, mt)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	switch len(matches) {
	case 0:
		return nil, fmt.Errorf("%w: task prefix %q", ErrNotFound, prefix)
	case 1:
		return matches[0], nil
	default:
		return nil, fmt.Errorf("db: ambiguous task prefix %q (%d matches)", prefix, len(matches))
	}
}

func ListTasksInBoard(ctx context.Context, q querier, boardID int64, columnName string) ([]*model.Task, error) {
	args := []any{boardID}
	where := "WHERE t.board_id = ?"
	if columnName != "" {
		where += " AND c.name = ?"
		args = append(args, columnName)
	}
	return runListTasks(ctx, q, taskSelect+" "+where+" ORDER BY c.position, t.position", args)
}

func ListTasksInColumn(ctx context.Context, q querier, columnID int64) ([]*model.Task, error) {
	return runListTasks(ctx, q, taskSelect+" WHERE t.column_id = ? ORDER BY t.position", []any{columnID})
}

func ListSubtasks(ctx context.Context, q querier, parentID int64) ([]*model.Task, error) {
	return runListTasks(ctx, q, taskSelect+" WHERE t.parent_id = ? ORDER BY t.position", []any{parentID})
}

func UpdateTask(ctx context.Context, q querier, id int64, patch TaskPatch) error {
	sets := []string{}
	args := []any{}
	if patch.Title != nil {
		sets = append(sets, "title = ?")
		args = append(args, *patch.Title)
	}
	if patch.Description != nil {
		sets = append(sets, "description = ?")
		args = append(args, *patch.Description)
	}
	if patch.Priority != nil {
		p, err := model.ParsePriority(*patch.Priority)
		if err != nil {
			return err
		}
		sets = append(sets, "priority = ?")
		args = append(args, p)
	}
	if patch.DueDate != nil {
		sets = append(sets, "due_date = ?")
		args = append(args, dbutil.FormatDueDate(*patch.DueDate))
	} else if patch.ClearDueDate {
		sets = append(sets, "due_date = ?")
		args = append(args, nil)
	}
	if patch.ParentID != nil {
		sets = append(sets, "parent_id = ?")
		args = append(args, *patch.ParentID)
	} else if patch.ClearParentID {
		sets = append(sets, "parent_id = ?")
		args = append(args, nil)
	}
	if len(sets) == 0 {
		return errors.New("db: UpdateTask called with no fields")
	}
	sets = append(sets, "updated_at = strftime('%Y-%m-%dT%H:%M:%fZ','now')")
	args = append(args, id)

	res, err := q.ExecContext(ctx,
		"UPDATE tasks SET "+strings.Join(sets, ", ")+" WHERE id = ?", args...)
	if err != nil {
		return fmt.Errorf("db: update task: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("%w: task %d", ErrNotFound, id)
	}
	return nil
}

func MoveTask(ctx context.Context, q querier, taskID, targetColumnID int64) error {
	var currentCol int64
	err := q.QueryRowContext(ctx,
		`SELECT column_id FROM tasks WHERE id = ?`, taskID,
	).Scan(&currentCol)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("%w: task %d", ErrNotFound, taskID)
		}
		return fmt.Errorf("db: get task column: %w", err)
	}
	if currentCol == targetColumnID {
		return nil
	}

	var targetBoard int64
	err = q.QueryRowContext(ctx,
		`SELECT board_id FROM columns WHERE id = ?`, targetColumnID,
	).Scan(&targetBoard)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("%w: column %d", ErrNotFound, targetColumnID)
		}
		return fmt.Errorf("db: get column: %w", err)
	}

	pos, err := computeNextPosition(ctx, q, targetColumnID, nil)
	if errors.Is(err, errGapTooSmall) {
		if _, rerr := RebalanceColumn(ctx, q, targetColumnID); rerr != nil {
			return fmt.Errorf("db: rebalance before move: %w", rerr)
		}
		pos, err = computeNextPosition(ctx, q, targetColumnID, nil)
	}
	if err != nil {
		return fmt.Errorf("db: compute position for move: %w", err)
	}

	_, err = q.ExecContext(ctx, `
		UPDATE tasks
		SET column_id = ?, position = ?, updated_at = strftime('%Y-%m-%dT%H:%M:%fZ','now')
		WHERE id = ?`,
		targetColumnID, pos, taskID,
	)
	if err != nil {
		return fmt.Errorf("db: move task: %w", err)
	}
	ensureHealthyPosition(ctx, q, targetColumnID)
	return nil
}

func DeleteTask(ctx context.Context, q querier, id int64) error {
	res, err := q.ExecContext(ctx, `DELETE FROM tasks WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("db: delete task: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("%w: task %d", ErrNotFound, id)
	}
	return nil
}

func runListTasks(ctx context.Context, q querier, query string, args []any) ([]*model.Task, error) {
	rows, err := q.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("db: list tasks: %w", err)
	}
	defer rows.Close()
	var out []*model.Task
	for rows.Next() {
		t, err := scanTask(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func scanTask(s rowScanner) (*model.Task, error) {
	var (
		t          model.Task
		parentID   sql.NullInt64
		dueDate    sql.NullString
		creAt, upd string
	)
	if err := s.Scan(
		&t.ID, &t.BoardID, &t.ColumnID, &t.ColumnName,
		&parentID, &t.Title, &t.Description, &t.Priority, &t.Position,
		&dueDate, &creAt, &upd,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%w: task", ErrNotFound)
		}
		return nil, fmt.Errorf("db: scan task: %w", err)
	}
	if parentID.Valid {
		v := parentID.Int64
		t.ParentID = &v
	}
	if dueDate.Valid && dueDate.String != "" {
		parsed, err := dbutil.ParseDueDate(dueDate.String)
		if err != nil {
			return nil, fmt.Errorf("db: parse due_date: %w", err)
		}
		t.DueDate = &parsed
	}
	var err error
	if t.CreatedAt, err = dbutil.ParseTime(creAt); err != nil {
		return nil, err
	}
	if t.UpdatedAt, err = dbutil.ParseTime(upd); err != nil {
		return nil, err
	}
	return &t, nil
}