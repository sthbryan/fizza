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
	Priority      *model.Priority
	DueDate       *time.Time
	ClearDueDate  bool
	ParentID      *int64
	ClearParentID bool
}

type TaskFilter struct {
	ColumnName string
	Priorities []model.Priority
	DueBefore  *time.Time
	DueAfter   *time.Time
	Search     string
}

func CreateTask(ctx context.Context, q querier, t *model.Task) error {
	if t.ColumnID == 0 {
		return errors.New("db: task.ColumnID required")
	}
	if t.Priority.IsZero() {
		t.Priority = model.Priority{Value: model.DefaultPriority}
	}
	if err := t.Validate(); err != nil {
		return err
	}
	if t.ParentID != nil {
		cycle, err := WouldCreateCycle(ctx, q, 0, *t.ParentID)
		if err != nil {
			return fmt.Errorf("db: check parent: %w", err)
		}
		if cycle {
			return model.ErrTaskCycle
		}
	}

	exec := q
	committed := false
	if txer, ok := q.(transactor); ok {
		tx, err := txer.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("db: begin tx: %w", err)
		}
		defer func() {
			if !committed {
				_ = tx.Rollback()
			}
		}()
		exec = tx
		if err := createTaskInsert(ctx, exec, t); err != nil {
			return err
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("db: commit create task: %w", err)
		}
		committed = true
	} else {
		if err := createTaskInsert(ctx, exec, t); err != nil {
			return err
		}
	}

	ensureHealthyPosition(ctx, q, t.ColumnID)

	fresh, err := GetTask(ctx, q, t.ID)
	if err != nil {
		return err
	}
	*t = *fresh
	return nil
}

func createTaskInsert(ctx context.Context, q querier, t *model.Task) error {
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
		t.BoardID, t.ColumnID, dbutil.NullableInt(t.ParentID), t.Title, t.Description, t.Priority.String(), pos, dueParam,
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

func ListTasksInBoard(ctx context.Context, q querier, boardID int64, filter TaskFilter) ([]*model.Task, error) {
	args := []any{boardID}
	where := "WHERE t.board_id = ?"
	if filter.ColumnName != "" {
		where += " AND c.name = ?"
		args = append(args, filter.ColumnName)
	}
	if len(filter.Priorities) > 0 {
		where += " AND t.priority IN (?" + strings.Repeat(",?", len(filter.Priorities)-1) + ")"
		for _, p := range filter.Priorities {
			args = append(args, p.String())
		}
	}
	if filter.DueBefore != nil {
		where += " AND t.due_date IS NOT NULL AND t.due_date <= ?"
		args = append(args, dbutil.FormatDueDate(*filter.DueBefore))
	}
	if filter.DueAfter != nil {
		where += " AND t.due_date IS NOT NULL AND t.due_date >= ?"
		args = append(args, dbutil.FormatDueDate(*filter.DueAfter))
	}
	if filter.Search != "" {
		where += " AND (t.title LIKE ? OR t.description LIKE ?)"
		needle := "%" + filter.Search + "%"
		args = append(args, needle, needle)
	}
	return runListTasks(ctx, q, taskSelect+" "+where+" ORDER BY c.position, t.position", args)
}

func ListTasksInColumn(ctx context.Context, q querier, columnID int64) ([]*model.Task, error) {
	return runListTasks(ctx, q, taskSelect+" WHERE t.column_id = ? ORDER BY t.position", []any{columnID})
}

func FirstTaskInColumn(ctx context.Context, q querier, columnID int64) (*model.Task, error) {
	row := q.QueryRowContext(ctx, taskSelect+" WHERE t.column_id = ? ORDER BY t.position LIMIT 1", columnID)
	return scanTask(row)
}

func NextTaskInColumn(ctx context.Context, q querier, columnID, afterID int64) (*model.Task, error) {
	row := q.QueryRowContext(ctx, taskSelect+" WHERE t.column_id = ? AND t.id > ? ORDER BY t.id LIMIT 1", columnID, afterID)
	return scanTask(row)
}

func ListSubtasks(ctx context.Context, q querier, parentID int64) ([]*model.Task, error) {
	return runListTasks(ctx, q, taskSelect+" WHERE t.parent_id = ? ORDER BY t.position", []any{parentID})
}

func WouldCreateCycle(ctx context.Context, q querier, taskID, proposedParent int64) (bool, error) {
	if taskID == proposedParent {
		return true, nil
	}
	current := proposedParent
	for i := 0; i < 10000; i++ {
		var parent sql.NullInt64
		err := q.QueryRowContext(ctx, `SELECT parent_id FROM tasks WHERE id = ?`, current).Scan(&parent)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return false, nil
			}
			return false, err
		}
		if !parent.Valid {
			return false, nil
		}
		if parent.Int64 == taskID {
			return true, nil
		}
		current = parent.Int64
	}
	return true, fmt.Errorf("db: parent chain too long (possible cycle)")
}

func UpdateTask(ctx context.Context, q querier, id int64, patch TaskPatch) error {
	sets := []string{}
	args := []any{}
	if patch.ParentID != nil {
		cycle, err := WouldCreateCycle(ctx, q, id, *patch.ParentID)
		if err != nil {
			return fmt.Errorf("db: check parent: %w", err)
		}
		if cycle {
			return model.ErrTaskCycle
		}
	}
	if patch.Title != nil {
		sets = append(sets, "title = ?")
		args = append(args, *patch.Title)
	}
	if patch.Description != nil {
		sets = append(sets, "description = ?")
		args = append(args, *patch.Description)
	}
	if patch.Priority != nil {
		sets = append(sets, "priority = ?")
		args = append(args, patch.Priority.String())
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
	return MoveTaskAt(ctx, q, taskID, targetColumnID, nil)
}

func MoveTaskAt(ctx context.Context, q querier, taskID, targetColumnID int64, beforeTaskID *int64) error {
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
	if currentCol == targetColumnID && beforeTaskID == nil {
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

	exec := q
	committed := false
	if txer, ok := q.(transactor); ok {
		tx, err := txer.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("db: begin tx: %w", err)
		}
		defer func() {
			if !committed {
				_ = tx.Rollback()
			}
		}()
		exec = tx
	}

	var pos float64
	if beforeTaskID != nil {
		pos, err = computePositionBefore(ctx, exec, targetColumnID, *beforeTaskID)
	} else {
		pos, err = computeNextPosition(ctx, exec, targetColumnID, nil)
	}
	if errors.Is(err, errGapTooSmall) {
		if _, rerr := RebalanceColumn(ctx, exec, targetColumnID); rerr != nil {
			return fmt.Errorf("db: rebalance before move: %w", rerr)
		}
		if beforeTaskID != nil {
			pos, err = computePositionBefore(ctx, exec, targetColumnID, *beforeTaskID)
		} else {
			pos, err = computeNextPosition(ctx, exec, targetColumnID, nil)
		}
	}
	if err != nil {
		return fmt.Errorf("db: compute position for move: %w", err)
	}

	_, err = exec.ExecContext(ctx, `
		UPDATE tasks
		SET column_id = ?, position = ?, updated_at = strftime('%Y-%m-%dT%H:%M:%fZ','now')
		WHERE id = ?`,
		targetColumnID, pos, taskID,
	)
	if err != nil {
		return fmt.Errorf("db: move task: %w", err)
	}

	if tx, ok := exec.(*sql.Tx); ok {
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("db: commit move task: %w", err)
		}
		committed = true
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
		prio       string
		dueDate    sql.NullString
		creAt, upd string
	)
	if err := s.Scan(
		&t.ID, &t.BoardID, &t.ColumnID, &t.ColumnName,
		&parentID, &t.Title, &t.Description, &prio, &t.Position,
		&dueDate, &creAt, &upd,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%w: task", ErrNotFound)
		}
		return nil, fmt.Errorf("db: scan task: %w", err)
	}
	parsedPrio, err := model.NewPriority(prio)
	if err != nil {
		return nil, fmt.Errorf("db: parse priority %q: %w", prio, err)
	}
	t.Priority = parsedPrio
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
	if t.CreatedAt, err = dbutil.ParseTime(creAt); err != nil {
		return nil, err
	}
	if t.UpdatedAt, err = dbutil.ParseTime(upd); err != nil {
		return nil, err
	}
	return &t, nil
}