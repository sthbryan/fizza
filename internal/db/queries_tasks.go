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
	       t.due_date, t.completed_at, t.archived_at, t.created_at, t.updated_at
	FROM tasks t
	JOIN columns c ON c.id = t.column_id`

type TaskPatch struct {
	Title         *string
	Description   *string
	Priority      *model.Priority
	DueDate       *dbutil.Time
	ClearDueDate  bool
	ParentID      *int64
	ClearParentID bool
}

type TaskFilter struct {
	ColumnName      string
	Priorities      []model.Priority
	DueBefore       *time.Time
	DueAfter        *time.Time
	Search          string
	Tags            []string
	IncludeDone     bool
	IncludeArchived bool
	OnlyArchived    bool
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
	if err := checkColumnWIP(ctx, q, t.ColumnID, 0); err != nil {
		return err
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

	projectID := boardProjectID(ctx, q, t.BoardID)
	_ = RecordEvent(ctx, q, Event{
		ProjectID: projectID,
		BoardID:   &t.BoardID,
		TaskID:    &t.ID,
		Kind:      "task_create",
		Payload:   t.Title,
	})
	return nil
}

func checkColumnWIP(ctx context.Context, q Querier, columnID, fromColumnID int64) error {
	if columnID == 0 || columnID == fromColumnID {
		return nil
	}
	var wipLimit *int64
	err := q.QueryRowContext(ctx,
		`SELECT wip_limit FROM columns WHERE id = ?`, columnID,
	).Scan(&wipLimit)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("%w: column %d", ErrNotFound, columnID)
		}
		return fmt.Errorf("db: get column wip: %w", err)
	}
	if wipLimit == nil {
		return nil
	}
	var count int64
	if err := q.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM tasks WHERE column_id = ? AND archived_at IS NULL`, columnID,
	).Scan(&count); err != nil {
		return fmt.Errorf("db: count tasks in column: %w", err)
	}
	if count >= *wipLimit {
		return ErrWIPLimitReached
	}
	return nil
}

func boardProjectID(ctx context.Context, q Querier, boardID int64) *int64 {
	var projectID int64
	if err := q.QueryRowContext(ctx,
		`SELECT project_id FROM boards WHERE id = ?`, boardID,
	).Scan(&projectID); err != nil {
		return nil
	}
	return &projectID
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
		dueParam = dbutil.FormatDueDate(t.DueDate.Time)
	}

	var completedParam any
	if colName, err := columnNameByID(ctx, q, t.ColumnID); err == nil && IsTerminalColumn(colName) {
		completedParam = time.Now().UTC().Format("2006-01-02T15:04:05.000000Z")
	}

	res, err := q.ExecContext(ctx, `
		INSERT INTO tasks
		  (board_id, column_id, parent_id, title, description, priority, position, due_date, completed_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		t.BoardID, t.ColumnID, dbutil.NullableInt(t.ParentID), t.Title, t.Description, t.Priority.String(), pos, dueParam, completedParam,
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
	var t model.Task
	err := q.GetContext(ctx, &t, taskSelect+` WHERE t.id = ?`, id)
	if err != nil {
		return nil, mapErrNotFound(err, fmt.Sprintf("task %d", id))
	}
	return &t, nil
}

func GetTaskByPrefix(ctx context.Context, q Querier, prefix string) (*model.Task, error) {
	prefix = strings.TrimSpace(prefix)
	if prefix == "" {
		return nil, errors.New("db: empty prefix")
	}
	if !dbutil.IsDigits(prefix) {
		return nil, fmt.Errorf("db: task prefix must be numeric: %q", prefix)
	}
	var matches []*model.Task
	err := q.SelectContext(ctx, &matches,
		taskSelect+` WHERE CAST(t.id AS TEXT) LIKE ? ORDER BY t.id LIMIT 2`,
		prefix+"%",
	)
	if err != nil {
		return nil, fmt.Errorf("db: prefix lookup: %w", err)
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

	if filter.OnlyArchived {
		where = append(where, "t.archived_at IS NOT NULL")
	} else if !filter.IncludeArchived {
		where = append(where, "t.archived_at IS NULL")
	}

	if filter.ColumnName != "" {
		where = append(where, "c.name = ?")
		args = append(args, filter.ColumnName)
	} else if !filter.IncludeDone && !filter.OnlyArchived {
		where = append(where, "NOT ("+terminalColumnSQL+")")
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
	order := " ORDER BY c.position, t.position"
	if filter.OnlyArchived {
		order = " ORDER BY t.archived_at DESC, t.id DESC"
	}
	query := strings.Replace(taskSelect, "SELECT", "SELECT "+distinct, 1) +
		" " + strings.Join(joins, " ") +
		" WHERE " + strings.Join(where, " AND ") +
		order
	expanded, expandedArgs, err := sqlx.In(query, args...)
	if err != nil {
		return nil, fmt.Errorf("db: expand IN clauses: %w", err)
	}
	expanded = q.Rebind(expanded)
	return runListTasks(ctx, q, expanded, expandedArgs)
}

func ListTasksInColumn(ctx context.Context, q Querier, columnID int64) ([]*model.Task, error) {
	return runListTasks(ctx, q,
		taskSelect+` WHERE t.column_id = ? AND t.archived_at IS NULL ORDER BY t.position`,
		[]any{columnID})
}

func CountTasksInColumn(ctx context.Context, q Querier, columnID int64) (int64, error) {
	var n int64
	err := q.GetContext(ctx, &n,
		`SELECT COUNT(*) FROM tasks WHERE column_id = ? AND archived_at IS NULL`, columnID)
	if err != nil {
		return 0, fmt.Errorf("db: count tasks in column: %w", err)
	}
	return n, nil
}

func CountArchivedInBoard(ctx context.Context, q Querier, boardID int64) (int64, error) {
	var n int64
	err := q.GetContext(ctx, &n,
		`SELECT COUNT(*) FROM tasks WHERE board_id = ? AND archived_at IS NOT NULL`, boardID)
	if err != nil {
		return 0, fmt.Errorf("db: count archived: %w", err)
	}
	return n, nil
}

func columnNameByID(ctx context.Context, q Querier, columnID int64) (string, error) {
	var name string
	err := q.QueryRowContext(ctx, `SELECT name FROM columns WHERE id = ?`, columnID).Scan(&name)
	if err != nil {
		return "", err
	}
	return name, nil
}

func FirstTaskInColumn(ctx context.Context, q Querier, columnID int64) (*model.Task, error) {
	var t model.Task
	err := q.GetContext(ctx, &t, taskSelect+" WHERE t.column_id = ? ORDER BY t.position LIMIT 1", columnID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("db: first task in column: %w", err)
	}
	return &t, nil
}

func NextTaskInColumn(ctx context.Context, q Querier, columnID, afterID int64) (*model.Task, error) {
	var t model.Task
	err := q.GetContext(ctx, &t, taskSelect+" WHERE t.column_id = ? AND t.id > ? ORDER BY t.id LIMIT 1", columnID, afterID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("db: next task in column: %w", err)
	}
	return &t, nil
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
		var parent *int64
		err := q.QueryRowContext(ctx, `SELECT parent_id FROM tasks WHERE id = ?`, current).Scan(&parent)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return false, nil
			}
			return false, err
		}
		if parent == nil {
			return false, nil
		}
		if *parent == taskID {
			return true, nil
		}
		current = *parent
	}
	return true, fmt.Errorf("db: parent chain too long (possible cycle)")
}

func (p TaskPatch) Empty() bool {
	return p.Title == nil && p.Description == nil && p.Priority == nil &&
		p.DueDate == nil && !p.ClearDueDate &&
		p.ParentID == nil && !p.ClearParentID
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
		args = append(args, dbutil.FormatDueDate(patch.DueDate.Time))
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

	if t, err := GetTask(ctx, q, id); err == nil {
		projectID := boardProjectID(ctx, q, t.BoardID)
		_ = RecordEvent(ctx, q, Event{
			ProjectID: projectID,
			BoardID:   &t.BoardID,
			TaskID:    &id,
			Kind:      "task_update",
		})
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
	var archived any
	err := q.QueryRowContext(ctx,
		`SELECT column_id, archived_at FROM tasks WHERE id = ?`, taskID,
	).Scan(&currentCol, &archived)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("%w: task %d", ErrNotFound, taskID)
		}
		return fmt.Errorf("db: get task column: %w", err)
	}
	if archived != nil {
		return fmt.Errorf("%w: task %d is archived; unarchive before moving", model.ErrValidation, taskID)
	}
	if currentCol == targetColumnID && beforeTaskID == nil {
		return nil
	}

	var targetBoard int64
	var targetColName string
	err = q.QueryRowContext(ctx,
		`SELECT board_id, name FROM columns WHERE id = ?`, targetColumnID,
	).Scan(&targetBoard, &targetColName)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("%w: column %d", ErrNotFound, targetColumnID)
		}
		return fmt.Errorf("db: get column: %w", err)
	}

	var currentColName string
	_ = q.QueryRowContext(ctx,
		`SELECT name FROM columns WHERE id = ?`, currentCol,
	).Scan(&currentColName)

	if !force {
		if err := checkColumnWIP(ctx, q, targetColumnID, currentCol); err != nil {
			return err
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

	toTerminal := IsTerminalColumn(targetColName)
	fromTerminal := IsTerminalColumn(currentColName)
	var completedSQL string
	switch {
	case toTerminal && !fromTerminal:
		completedSQL = ", completed_at = strftime('%Y-%m-%dT%H:%M:%fZ','now')"
	case !toTerminal && fromTerminal:
		completedSQL = ", completed_at = NULL"
	}

	_, err = exec.ExecContext(ctx, `
		UPDATE tasks
		SET column_id = ?, position = ?, updated_at = strftime('%Y-%m-%dT%H:%M:%fZ','now')`+completedSQL+`
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

	projectID := boardProjectID(ctx, q, targetBoard)
	taskIDCopy := taskID
	boardIDCopy := targetBoard
	_ = RecordEvent(ctx, q, Event{
		ProjectID: projectID,
		BoardID:   &boardIDCopy,
		TaskID:    &taskIDCopy,
		Kind:      "task_move",
	})
	return nil
}

func ArchiveTask(ctx context.Context, q Querier, id int64) error {
	t, err := GetTask(ctx, q, id)
	if err != nil {
		return err
	}
	if t.ArchivedAt != nil {
		return nil
	}
	res, err := q.ExecContext(ctx, `
		UPDATE tasks
		SET archived_at = strftime('%Y-%m-%dT%H:%M:%fZ','now'),
		    updated_at = strftime('%Y-%m-%dT%H:%M:%fZ','now')
		WHERE id = ? AND archived_at IS NULL`, id)
	if err != nil {
		return fmt.Errorf("db: archive task: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("%w: task %d", ErrNotFound, id)
	}
	projectID := boardProjectID(ctx, q, t.BoardID)
	boardID := t.BoardID
	taskID := id
	_ = RecordEvent(ctx, q, Event{
		ProjectID: projectID,
		BoardID:   &boardID,
		TaskID:    &taskID,
		Kind:      "task_update",
		Payload:   "archive",
	})
	return nil
}

func UnarchiveTask(ctx context.Context, q Querier, id int64) error {
	t, err := GetTask(ctx, q, id)
	if err != nil {
		return err
	}
	if t.ArchivedAt == nil {
		return nil
	}
	res, err := q.ExecContext(ctx, `
		UPDATE tasks
		SET archived_at = NULL,
		    updated_at = strftime('%Y-%m-%dT%H:%M:%fZ','now')
		WHERE id = ? AND archived_at IS NOT NULL`, id)
	if err != nil {
		return fmt.Errorf("db: unarchive task: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("%w: task %d", ErrNotFound, id)
	}
	projectID := boardProjectID(ctx, q, t.BoardID)
	boardID := t.BoardID
	taskID := id
	_ = RecordEvent(ctx, q, Event{
		ProjectID: projectID,
		BoardID:   &boardID,
		TaskID:    &taskID,
		Kind:      "task_update",
		Payload:   "unarchive",
	})
	return nil
}

func ArchiveDoneInBoard(ctx context.Context, q Querier, boardID int64) (int64, error) {
	res, err := q.ExecContext(ctx, `
		UPDATE tasks
		SET archived_at = strftime('%Y-%m-%dT%H:%M:%fZ','now'),
		    updated_at = strftime('%Y-%m-%dT%H:%M:%fZ','now')
		WHERE board_id = ?
		  AND archived_at IS NULL
		  AND column_id IN (
		    SELECT id FROM columns c
		    WHERE c.board_id = ? AND `+terminalColumnSQL+`
		  )`, boardID, boardID)
	if err != nil {
		return 0, fmt.Errorf("db: archive done in board: %w", err)
	}
	n, _ := res.RowsAffected()
	if n > 0 {
		projectID := boardProjectID(ctx, q, boardID)
		boardIDCopy := boardID
		_ = RecordEvent(ctx, q, Event{
			ProjectID: projectID,
			BoardID:   &boardIDCopy,
			Kind:      "task_update",
			Payload:   fmt.Sprintf("archive_done:%d", n),
		})
	}
	return n, nil
}

func DeleteTask(ctx context.Context, q Querier, id int64) error {
	t, getErr := GetTask(ctx, q, id)
	res, err := q.ExecContext(ctx, `DELETE FROM tasks WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("db: delete task: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("%w: task %d", ErrNotFound, id)
	}
	if getErr == nil && t != nil {
		projectID := boardProjectID(ctx, q, t.BoardID)
		boardIDCopy := t.BoardID
		_ = RecordEvent(ctx, q, Event{
			ProjectID: projectID,
			BoardID:   &boardIDCopy,
			Kind:      "task_delete",
			Payload:   fmt.Sprintf("%d:%s", id, t.Title),
		})
	}
	return nil
}

func runListTasks(ctx context.Context, q Querier, query string, args []any) ([]*model.Task, error) {
	var out []*model.Task
	if err := q.SelectContext(ctx, &out, query, args...); err != nil {
		return nil, fmt.Errorf("db: list tasks: %w", err)
	}
	return out, nil
}
