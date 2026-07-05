package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/fizza/fizza/internal/db"
	"github.com/fizza/fizza/internal/dbutil"
	"github.com/fizza/fizza/internal/model"
)

type Service struct {
	conn       *sql.DB
	project    string
	board      string
	column     string
	resolved   *Resolved
}

type Resolved struct {
	Project *model.Project
	Board   *model.Board
	Column  *model.Column
}

func New(conn *sql.DB, project, board, column string) *Service {
	return &Service{conn: conn, project: project, board: board, column: column}
}

func (s *Service) DB() *sql.DB { return s.conn }

func (s *Service) Close() error { return s.conn.Close() }

func (s *Service) Resolve(ctx context.Context) (*Resolved, error) {
	if s.resolved != nil {
		return s.resolved, nil
	}
	r := &Resolved{}
	if s.project != "" {
		p, err := db.GetProjectByName(ctx, s.conn, s.project)
		if err != nil {
			return nil, err
		}
		r.Project = p
	}
	if s.board != "" {
		if r.Project == nil {
			return nil, errors.New("service: board requires project")
		}
		b, err := findBoardByName(ctx, s.conn, r.Project.ID, s.board)
		if err != nil {
			return nil, err
		}
		r.Board = b
	}
	if s.column != "" {
		if r.Board == nil {
			return nil, errors.New("service: column requires board")
		}
		c, err := db.GetColumnByName(ctx, s.conn, r.Board.ID, s.column)
		if err != nil {
			return nil, err
		}
		r.Column = c
	}
	s.resolved = r
	return r, nil
}

func (s *Service) ResolveProject(ctx context.Context) (*model.Project, error) {
	r, err := s.Resolve(ctx)
	if err != nil {
		return nil, err
	}
	if r.Project == nil {
		return nil, fmt.Errorf("%w: no default project set (run `fizza project set <name>` first)", ErrValidation)
	}
	return r.Project, nil
}

func (s *Service) ResolveBoard(ctx context.Context) (*Resolved, error) {
	r, err := s.Resolve(ctx)
	if err != nil {
		return nil, err
	}
	if r.Project == nil {
		return nil, fmt.Errorf("%w: no default project set", ErrValidation)
	}
	if r.Board == nil {
		return nil, fmt.Errorf("%w: board %q in project %q", db.ErrNotFound, s.board, s.project)
	}
	return r, nil
}

func (s *Service) ResolveColumn(ctx context.Context, defaultToFirst bool) (*Resolved, error) {
	r, err := s.ResolveBoard(ctx)
	if err != nil {
		return nil, err
	}
	if r.Column != nil {
		return r, nil
	}
	cols, err := db.ListColumns(ctx, s.conn, r.Board.ID)
	if err != nil {
		return nil, err
	}
	if len(cols) == 0 {
		return nil, fmt.Errorf("%w: board has no columns", db.ErrNotFound)
	}
	if defaultToFirst {
		r.Column = cols[0]
		return r, nil
	}
	return nil, fmt.Errorf("%w: column %q in board %q", db.ErrNotFound, s.column, s.board)
}

var ErrValidation = errors.New("validation")

func findBoardByName(ctx context.Context, conn *sql.DB, projectID int64, name string) (*model.Board, error) {
	boards, err := db.ListBoards(ctx, conn, projectID)
	if err != nil {
		return nil, err
	}
	for _, b := range boards {
		if b.Name == name {
			return b, nil
		}
	}
	return nil, fmt.Errorf("%w: board %q in project %d", db.ErrNotFound, name, projectID)
}

type ProjectCounts struct {
	Projects int64 `json:"projects"`
	Boards   int64 `json:"boards"`
	Tasks    int64 `json:"tasks"`
}

func (s *Service) ProjectCounts(ctx context.Context) (ProjectCounts, error) {
	var c ProjectCounts
	row := s.conn.QueryRowContext(ctx,
		`SELECT
			(SELECT COUNT(*) FROM projects),
			(SELECT COUNT(*) FROM boards),
			(SELECT COUNT(*) FROM tasks)`)
	if err := row.Scan(&c.Projects, &c.Boards, &c.Tasks); err != nil {
		return c, err
	}
	return c, nil
}

type TaskCreateInput struct {
	Title       string
	Description string
	Priority    model.Priority
	DueDate     *time.Time
	ParentID    *int64
	ColumnID    int64
}

func (s *Service) CreateTask(ctx context.Context, in TaskCreateInput) (*model.Task, error) {
	r, err := s.ResolveBoard(ctx)
	if err != nil {
		return nil, err
	}
	col := in.ColumnID
	if col == 0 {
		if r.Column != nil {
			col = r.Column.ID
		} else {
			cols, err := db.ListColumns(ctx, s.conn, r.Board.ID)
			if err != nil {
				return nil, err
			}
			if len(cols) == 0 {
				return nil, fmt.Errorf("%w: board has no columns", db.ErrNotFound)
			}
			col = cols[0].ID
		}
	}
	if in.Priority.IsZero() {
		in.Priority = model.Priority{Value: model.DefaultPriority}
	}
	t := &model.Task{
		BoardID:     r.Board.ID,
		ColumnID:    col,
		Title:       strings.TrimSpace(in.Title),
		Description: in.Description,
		Priority:    in.Priority,
		DueDate:     in.DueDate,
		ParentID:    in.ParentID,
	}
	if err := db.CreateTask(ctx, s.conn, t); err != nil {
		return nil, err
	}
	return t, nil
}

func (s *Service) ListTasks(ctx context.Context, filter db.TaskFilter) ([]*model.Task, error) {
	r, err := s.ResolveBoard(ctx)
	if err != nil {
		return nil, err
	}
	return db.ListTasksInBoard(ctx, s.conn, r.Board.ID, filter)
}

func (s *Service) MoveTask(ctx context.Context, taskID, targetColumnID int64, beforeTaskID *int64) error {
	return db.MoveTaskAt(ctx, s.conn, taskID, targetColumnID, beforeTaskID)
}

func (s *Service) UpdateTask(ctx context.Context, id int64, patch db.TaskPatch) (*model.Task, error) {
	if err := db.UpdateTask(ctx, s.conn, id, patch); err != nil {
		return nil, err
	}
	return db.GetTask(ctx, s.conn, id)
}

func (s *Service) DeleteTask(ctx context.Context, id int64) error {
	return db.DeleteTask(ctx, s.conn, id)
}

func (s *Service) GetTask(ctx context.Context, id int64) (*model.Task, error) {
	return db.GetTask(ctx, s.conn, id)
}

func (s *Service) GetTaskByPrefix(ctx context.Context, prefix string) (*model.Task, error) {
	return db.GetTaskByPrefix(ctx, s.conn, prefix)
}

func ParseDue(s string) (*time.Time, error) {
	if s == "" {
		return nil, nil
	}
	t, err := dbutil.ParseDueDate(s)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func ParseInt64Flexible(s string) (int64, error) {
	if s == "" {
		return 0, fmt.Errorf("empty")
	}
	var n int64
	for _, r := range s {
		if r < '0' || r > '9' {
			return 0, fmt.Errorf("not numeric: %q", s)
		}
		n = n*10 + int64(r-'0')
		if n > 1<<62 {
			return 0, fmt.Errorf("too large: %q", s)
		}
	}
	return n, nil
}

func SplitColumns(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}