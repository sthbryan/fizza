package db

import (
	"context"
	"fmt"
	"strings"

	"github.com/fizza/fizza/internal/model"
)

const doneColumnSQL = `lower(c.name) IN ('done', 'completed', 'closed')`

const overdueSQL = `t.due_date IS NOT NULL
	AND t.due_date < strftime('%Y-%m-%dT%H:%M:%fZ','now')
	AND NOT (` + doneColumnSQL + `)`

type statsScopeIDs struct {
	projectID *int64
	boardID   *int64
	project   string
	board     string
}

func resolveStatsScope(ctx context.Context, q Querier, projectName, boardName string) (statsScopeIDs, error) {
	var s statsScopeIDs
	projectName = strings.TrimSpace(projectName)
	boardName = strings.TrimSpace(boardName)
	if projectName == "" {
		if boardName != "" {
			return s, fmt.Errorf("%w: board filter requires project", model.ErrValidation)
		}
		return s, nil
	}
	p, err := GetProjectByName(ctx, q, projectName)
	if err != nil {
		return s, err
	}
	s.projectID = &p.ID
	s.project = p.Name
	if boardName == "" {
		return s, nil
	}
	b, err := GetBoardByName(ctx, q, p.ID, boardName)
	if err != nil {
		return s, err
	}
	s.boardID = &b.ID
	s.board = b.Name
	return s, nil
}

func (s statsScopeIDs) taskWhere(aliasT, aliasB, aliasP string) (string, []any) {
	var parts []string
	var args []any
	if s.projectID != nil {
		parts = append(parts, aliasP+".id = ?")
		args = append(args, *s.projectID)
	}
	if s.boardID != nil {
		parts = append(parts, aliasT+".board_id = ?")
		args = append(args, *s.boardID)
	}
	_ = aliasB
	if len(parts) == 0 {
		return "1=1", nil
	}
	return strings.Join(parts, " AND "), args
}

func GetStats(ctx context.Context, q Querier, projectName, boardName string) (*model.Stats, error) {
	scope, err := resolveStatsScope(ctx, q, projectName, boardName)
	if err != nil {
		return nil, err
	}

	out := &model.Stats{
		Scope: model.StatsScope{
			Project: scope.project,
			Board:   scope.board,
		},
		ByPriority:    []model.NamedCount{},
		ByColumn:      []model.NamedCount{},
		CreatedByDay:  []model.DayCount{},
		ActivityByDay: []model.DayCount{},
	}

	if err := fillStatsTotals(ctx, q, scope, out); err != nil {
		return nil, err
	}
	if err := fillStatsByPriority(ctx, q, scope, out); err != nil {
		return nil, err
	}
	if err := fillStatsByColumn(ctx, q, scope, out); err != nil {
		return nil, err
	}
	if err := fillStatsCreatedByDay(ctx, q, scope, out); err != nil {
		return nil, err
	}
	if err := fillStatsActivityByDay(ctx, q, scope, out); err != nil {
		return nil, err
	}

	if scope.projectID == nil {
		rows, err := listProjectStats(ctx, q)
		if err != nil {
			return nil, err
		}
		out.ByProject = rows
	}
	if scope.boardID == nil {
		rows, err := listBoardStats(ctx, q, scope)
		if err != nil {
			return nil, err
		}
		out.ByBoard = rows
	}

	return out, nil
}

func fillStatsTotals(ctx context.Context, q Querier, scope statsScopeIDs, out *model.Stats) error {

	switch {
	case scope.boardID != nil:
		out.Totals.Projects = 1
		out.Totals.Boards = 1
	case scope.projectID != nil:
		out.Totals.Projects = 1
		var n int64
		if err := q.GetContext(ctx, &n,
			`SELECT COUNT(*) FROM boards WHERE project_id = ?`, *scope.projectID); err != nil {
			return fmt.Errorf("db: stats boards count: %w", err)
		}
		out.Totals.Boards = n
	default:
		var n int64
		if err := q.GetContext(ctx, &n, `SELECT COUNT(*) FROM projects`); err != nil {
			return fmt.Errorf("db: stats projects count: %w", err)
		}
		out.Totals.Projects = n
		if err := q.GetContext(ctx, &n, `SELECT COUNT(*) FROM boards`); err != nil {
			return fmt.Errorf("db: stats boards count: %w", err)
		}
		out.Totals.Boards = n
	}

	where, args := scope.taskWhere("t", "b", "p")
	query := `
		SELECT
			COUNT(*) AS tasks,
			COALESCE(SUM(CASE WHEN ` + doneColumnSQL + ` THEN 1 ELSE 0 END), 0) AS done,
			COALESCE(SUM(CASE WHEN NOT (` + doneColumnSQL + `) THEN 1 ELSE 0 END), 0) AS open,
			COALESCE(SUM(CASE WHEN ` + overdueSQL + ` THEN 1 ELSE 0 END), 0) AS overdue
		FROM tasks t
		JOIN columns c ON c.id = t.column_id
		JOIN boards b ON b.id = t.board_id
		JOIN projects p ON p.id = b.project_id
		WHERE ` + where

	var row struct {
		Tasks   int64 `db:"tasks"`
		Done    int64 `db:"done"`
		Open    int64 `db:"open"`
		Overdue int64 `db:"overdue"`
	}
	if err := q.GetContext(ctx, &row, query, args...); err != nil {
		return fmt.Errorf("db: stats task totals: %w", err)
	}
	out.Totals.Tasks = row.Tasks
	out.Totals.Done = row.Done
	out.Totals.Open = row.Open
	out.Totals.Overdue = row.Overdue
	return nil
}

func fillStatsByPriority(ctx context.Context, q Querier, scope statsScopeIDs, out *model.Stats) error {
	where, args := scope.taskWhere("t", "b", "p")

	query := `
		SELECT pri.name AS name, COALESCE(cnt.count, 0) AS count
		FROM (
			SELECT 'low' AS name, 1 AS ord UNION ALL
			SELECT 'medium', 2 UNION ALL
			SELECT 'high', 3 UNION ALL
			SELECT 'urgent', 4
		) pri
		LEFT JOIN (
			SELECT t.priority AS name, COUNT(*) AS count
			FROM tasks t
			JOIN boards b ON b.id = t.board_id
			JOIN projects p ON p.id = b.project_id
			WHERE ` + where + `
			GROUP BY t.priority
		) cnt ON cnt.name = pri.name
		ORDER BY pri.ord`

	var rows []model.NamedCount
	if err := q.SelectContext(ctx, &rows, query, args...); err != nil {
		return fmt.Errorf("db: stats by priority: %w", err)
	}
	if rows == nil {
		rows = []model.NamedCount{}
	}
	out.ByPriority = rows
	return nil
}

func fillStatsByColumn(ctx context.Context, q Querier, scope statsScopeIDs, out *model.Stats) error {
	where, args := scope.taskWhere("t", "b", "p")
	query := `
		SELECT c.name AS name, COUNT(*) AS count
		FROM tasks t
		JOIN columns c ON c.id = t.column_id
		JOIN boards b ON b.id = t.board_id
		JOIN projects p ON p.id = b.project_id
		WHERE ` + where + `
		GROUP BY lower(c.name)
		ORDER BY COUNT(*) DESC, c.name ASC`

	var rows []model.NamedCount
	if err := q.SelectContext(ctx, &rows, query, args...); err != nil {
		return fmt.Errorf("db: stats by column: %w", err)
	}
	if rows == nil {
		rows = []model.NamedCount{}
	}
	out.ByColumn = rows
	return nil
}

func fillStatsCreatedByDay(ctx context.Context, q Querier, scope statsScopeIDs, out *model.Stats) error {
	where, args := scope.taskWhere("t", "b", "p")
	query := `
		SELECT date(t.created_at) AS date, COUNT(*) AS count
		FROM tasks t
		JOIN boards b ON b.id = t.board_id
		JOIN projects p ON p.id = b.project_id
		WHERE ` + where + `
		  AND t.created_at >= datetime('now', '-29 days')
		GROUP BY date(t.created_at)
		ORDER BY date ASC`

	var rows []model.DayCount
	if err := q.SelectContext(ctx, &rows, query, args...); err != nil {
		return fmt.Errorf("db: stats created by day: %w", err)
	}
	if rows == nil {
		rows = []model.DayCount{}
	}
	out.CreatedByDay = rows
	return nil
}

func fillStatsActivityByDay(ctx context.Context, q Querier, scope statsScopeIDs, out *model.Stats) error {
	var parts []string
	var args []any
	parts = append(parts, `e.created_at >= datetime('now', '-29 days')`)
	parts = append(parts, `e.kind IN ('task_create','task_update','task_move','task_delete')`)
	if scope.projectID != nil {
		parts = append(parts, `e.project_id = ?`)
		args = append(args, *scope.projectID)
	}
	if scope.boardID != nil {
		parts = append(parts, `e.board_id = ?`)
		args = append(args, *scope.boardID)
	}
	query := `
		SELECT date(e.created_at) AS date, COUNT(*) AS count
		FROM events e
		WHERE ` + strings.Join(parts, " AND ") + `
		GROUP BY date(e.created_at)
		ORDER BY date ASC`

	var rows []model.DayCount
	if err := q.SelectContext(ctx, &rows, query, args...); err != nil {
		return fmt.Errorf("db: stats activity by day: %w", err)
	}
	if rows == nil {
		rows = []model.DayCount{}
	}
	out.ActivityByDay = rows
	return nil
}

func listProjectStats(ctx context.Context, q Querier) ([]model.ProjectStatsRow, error) {
	query := `
		SELECT
			p.name AS name,
			(SELECT COUNT(*) FROM boards b WHERE b.project_id = p.id) AS boards,
			COUNT(t.id) AS tasks,
			COALESCE(SUM(CASE WHEN t.id IS NOT NULL AND ` + doneColumnSQL + ` THEN 1 ELSE 0 END), 0) AS done,
			COALESCE(SUM(CASE WHEN t.id IS NOT NULL AND NOT (` + doneColumnSQL + `) THEN 1 ELSE 0 END), 0) AS open,
			COALESCE(SUM(CASE WHEN t.id IS NOT NULL AND ` + overdueSQL + ` THEN 1 ELSE 0 END), 0) AS overdue
		FROM projects p
		LEFT JOIN boards b ON b.project_id = p.id
		LEFT JOIN tasks t ON t.board_id = b.id
		LEFT JOIN columns c ON c.id = t.column_id
		GROUP BY p.id
		ORDER BY p.name ASC`

	var rows []model.ProjectStatsRow
	if err := q.SelectContext(ctx, &rows, query); err != nil {
		return nil, fmt.Errorf("db: stats by project: %w", err)
	}
	if rows == nil {
		rows = []model.ProjectStatsRow{}
	}
	return rows, nil
}

func listBoardStats(ctx context.Context, q Querier, scope statsScopeIDs) ([]model.BoardStatsRow, error) {
	var parts []string
	var args []any
	if scope.projectID != nil {
		parts = append(parts, `p.id = ?`)
		args = append(args, *scope.projectID)
	}
	where := "1=1"
	if len(parts) > 0 {
		where = strings.Join(parts, " AND ")
	}
	query := `
		SELECT
			p.name AS project,
			b.name AS name,
			COUNT(t.id) AS tasks,
			COALESCE(SUM(CASE WHEN t.id IS NOT NULL AND ` + doneColumnSQL + ` THEN 1 ELSE 0 END), 0) AS done,
			COALESCE(SUM(CASE WHEN t.id IS NOT NULL AND NOT (` + doneColumnSQL + `) THEN 1 ELSE 0 END), 0) AS open,
			COALESCE(SUM(CASE WHEN t.id IS NOT NULL AND ` + overdueSQL + ` THEN 1 ELSE 0 END), 0) AS overdue
		FROM boards b
		JOIN projects p ON p.id = b.project_id
		LEFT JOIN tasks t ON t.board_id = b.id
		LEFT JOIN columns c ON c.id = t.column_id
		WHERE ` + where + `
		GROUP BY b.id
		ORDER BY p.name ASC, b.name ASC`

	var rows []model.BoardStatsRow
	if err := q.SelectContext(ctx, &rows, query, args...); err != nil {
		return nil, fmt.Errorf("db: stats by board: %w", err)
	}
	if rows == nil {
		rows = []model.BoardStatsRow{}
	}
	return rows, nil
}
