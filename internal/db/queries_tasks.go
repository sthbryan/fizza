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
	"github.com/jmoiron/sqlx"
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
	Tags       []string
}

func CreateTask(ctx context.Context, q Querier, t *model.Task) error {
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
	if txer, ok := q.(Transactor); ok {
		tx, err := txer.BeginTxx(ctx, nil)
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

func createTaskInsert(ctx context.Context, q Querier, t *model.Task) error {
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

func GetTask(ctx context.Context, q Querier, id int64) (*model.Task, error) {
	row := q.QueryRowContext(ctx, taskSelect+` WHERE t.id = ?`, id)
	return scanTask(row)
}

func GetTaskByPrefix(ctx context.Context, q Querier, prefix string) (*model.Task, error) {
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

func ListTasksInBoard(ctx context.Context, q Querier, boardID int64, filter TaskFilter) ([]*model.Task, error) {
	var (
		where    []string
		args     []any
		distinct string
		joins    []string
	)
	where = append(where, "t.board_id = ?")
	args = append(args, boardID)
	if filter.ColumnName != "" {
		where = append(where, "c.name = ?")
		args = append(args, filter.ColumnName)
	}
	if len(filter.Priorities) > 0 {
		priVals := make([]string, len(filter.Priorities))
		for i, p := range filter.Priorities {
			priVals[i] = p.String()
		}
		where = append(where, "t.priority IN (?)")
		args = append(args, priVals)
	}
	if filter.DueBefore != nil {
		where = append(where, "t.due_date IS NOT NULL AND t.due_date <= ?")
		args = append(args, dbutil.FormatDueDate(*filter.DueBefore))
	}
	if filter.DueAfter != nil {
		where = append(where, "t.due_date IS NOT NULL AND t.due_date >= ?")
		args = append(args, dbutil.FormatDueDate(*filter.DueAfter))
	}
	if filter.Search != "" {
		where = append(where, "(t.title LIKE ? OR t.description LIKE ?)")
		needle := "%" + filter.Search + "%"
		args = append(args, needle, needle)
	}
	if len(filter.Tags) > 0 {
		where = append(where, "tags.name IN (?)")
		args = append(args, filter.Tags)
		joins = append(joins, "JOIN task_tags ON task_tags.task_id = t.id JOIN tags ON tags.id = task_tags.tag_id")
		distinct = "DISTINCT "
	}
	query := strings.Replace(taskSelect, "SELECT", "SELECT "+distinct, 1) +
		" " + strings.Join(joins, " ") +
		" WHERE " + strings.Join(where, " AND ") +
		" ORDER BY c.position, t.position"
	expanded, expandedArgs, err := sqlx.In(query, args...)
	if err != nil {
		return nil, fmt.Errorf("db: expand IN clauses: %w", err)
	}
	expanded = q.Rebind(expanded)
	return runListTasks(ctx, q, expanded, expandedArgs)
}

func ListTasksInColumn(ctx context.Context, q Querier, columnID int64) ([]*model.Task, error) {
	return runListTasks(ctx, q, taskSelect+" WHERE t.column_id = ? ORDER BY t.position", []any{columnID})
}

func FirstTaskInColumn(ctx context.Context, q Querier, columnID int64) (*model.Task, error) {
	row := q.QueryRowContext(ctx, taskSelect+" WHERE t.column_id = ? ORDER BY t.position LIMIT 1", columnID)
	return scanTask(row)
}

func NextTaskInColumn(ctx context.Context, q Querier, columnID, afterID int64) (*model.Task, error) {
	row := q.QueryRowContext(ctx, taskSelect+" WHERE t.column_id = ? AND t.id > ? ORDER BY t.id LIMIT 1", columnID, afterID)
	return scanTask(row)
}

func ListSubtasks(ctx context.Context, q Querier, parentID int64) ([]*model.Task, error) {
	return runListTasks(ctx, q, taskSelect+" WHERE t.parent_id = ? ORDER BY t.position", []any{parentID})
}

func WouldCreateCycle(ctx context.Context, q Querier, taskID, proposedParent int64) (bool, error) {
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

func UpdateTask(ctx context.Context, q Querier, id int64, patch TaskPatch) error {
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

func MoveTask(ctx context.Context, q Querier, taskID, targetColumnID int64) error {
	return MoveTaskAt(ctx, q, taskID, targetColumnID, nil)
}

func MoveTaskForce(ctx context.Context, q Querier, taskID, targetColumnID int64, beforeTaskID *int64) error {
	return moveTaskAt(ctx, q, taskID, targetColumnID, beforeTaskID, true)
}

func MoveTaskAt(ctx context.Context, q Querier, taskID, targetColumnID int64, beforeTaskID *int64) error {
	return moveTaskAt(ctx, q, taskID, targetColumnID, beforeTaskID, false)
}

func moveTaskAt(ctx context.Context, q Querier, taskID, targetColumnID int64, beforeTaskID *int64, force bool) error {
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
	var wipLimit sql.NullInt64
	err = q.QueryRowContext(ctx,
		`SELECT board_id, wip_limit FROM columns WHERE id = ?`, targetColumnID,
	).Scan(&targetBoard, &wipLimit)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("%w: column %d", ErrNotFound, targetColumnID)
		}
		return fmt.Errorf("db: get column: %w", err)
	}

	if !force && wipLimit.Valid && currentCol != targetColumnID {
		var count int64
		if err := q.QueryRowContext(ctx,
			`SELECT COUNT(*) FROM tasks WHERE column_id = ?`, targetColumnID,
		).Scan(&count); err != nil {
			return fmt.Errorf("db: count tasks in target column: %w", err)
		}
		if count >= wipLimit.Int64 {
			return ErrWIPLimitReached
		}
	}

	exec := q
	committed := false
	if txer, ok := q.(Transactor); ok {
		tx, err := txer.BeginTxx(ctx, nil)
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

	if tx, ok := exec.(*sqlx.Tx); ok {
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("db: commit move task: %w", err)
		}
		committed = true
	}
	ensureHealthyPosition(ctx, q, targetColumnID)
	return nil
}

func DeleteTask(ctx context.Context, q Querier, id int64) error {
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

func runListTasks(ctx context.Context, q Querier, query string, args []any) ([]*model.Task, error) {
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
	if t.CreatedAt, err = parseTimeAsDBUtil(creAt); err != nil {
		return nil, err
	}
	if t.UpdatedAt, err = parseTimeAsDBUtil(upd); err != nil {
		return nil, err
	}
	return &t, nil
}