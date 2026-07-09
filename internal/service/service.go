package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/fizza/fizza/internal/db"
	"github.com/fizza/fizza/internal/dbutil"
	"github.com/fizza/fizza/internal/model"
	"github.com/jmoiron/sqlx"
)

type Service struct {
	pool     *db.Pool
	project  string
	board    string
	column   string
	resolved *Resolved
}

type Resolved struct {
	Project *model.Project
	Board   *model.Board
	Column  *model.Column
}

func New(conn *sqlx.DB, project, board, column string) *Service {
	return &Service{pool: db.NewSinglePool(conn), project: project, board: board, column: column}
}

func NewWithPool(pool *db.Pool, project, board, column string) *Service {
	return &Service{pool: pool, project: project, board: board, column: column}
}

func (s *Service) DB() *sqlx.DB { return s.pool.Write }

func (s *Service) Reader() *sqlx.DB { return s.pool.Read }

func (s *Service) Pool() *db.Pool { return s.pool }

func (s *Service) Close() error { return s.pool.Close() }

func (s *Service) Resolve(ctx context.Context) (*Resolved, error) {
	if s.resolved != nil {
		return s.resolved, nil
	}
	r := &Resolved{}
	if s.project != "" {
		p, err := db.GetProjectByName(ctx, s.pool.Write, s.project)
		if err != nil {
			return nil, err
		}
		r.Project = p
	}
	if s.board != "" {
		if r.Project == nil {
			return nil, errors.New("service: board requires project")
		}
		b, err := findBoardByName(ctx, s.pool.Write, r.Project.ID, s.board)
		if err != nil {
			return nil, err
		}
		r.Board = b
	} else if r.Project != nil {
		boards, err := db.ListBoards(ctx, s.pool.Write, r.Project.ID)
		if err != nil {
			return nil, err
		}
		if len(boards) == 1 {
			r.Board = boards[0]
		} else if len(boards) > 1 {
			return nil, fmt.Errorf("%w: project %q has %d boards; set a default board with `fizza config set board <name>` or pass --board",
				ErrValidation, r.Project.Name, len(boards))
		}
	}
	if s.column != "" {
		if r.Board == nil {
			return nil, errors.New("service: column requires board")
		}
		c, err := db.GetColumnByName(ctx, s.pool.Write, r.Board.ID, s.column)
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
		return nil, fmt.Errorf("%w: no default project set (run `fizza config set project <name>` first)", ErrValidation)
	}
	return r.Project, nil
}

func (s *Service) ResolveBoard(ctx context.Context) (*Resolved, error) {
	r, err := s.Resolve(ctx)
	if err != nil {
		return nil, err
	}
	if r.Project == nil {
		return nil, fmt.Errorf("%w: no default project set (run `fizza config set project <name>`)", ErrValidation)
	}
	if r.Board == nil {
		if s.board != "" {
			return nil, fmt.Errorf("%w: board %q in project %q", db.ErrNotFound, s.board, s.project)
		}
		return nil, fmt.Errorf("%w: no default board in project %q (run `fizza config set board <name>`)", ErrValidation, r.Project.Name)
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
	cols, err := db.ListColumns(ctx, s.pool.Write, r.Board.ID)
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

var ErrValidation = model.ErrValidation

func findBoardByName(ctx context.Context, conn *sqlx.DB, projectID int64, name string) (*model.Board, error) {
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
	row := s.pool.Write.QueryRowContext(ctx,
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
	DueDate     *dbutil.Time
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
			cols, err := db.ListColumns(ctx, s.pool.Write, r.Board.ID)
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
	if err := db.CreateTask(ctx, s.pool.Write, t); err != nil {
		return nil, err
	}
	return t, nil
}

func (s *Service) ListTasks(ctx context.Context, filter db.TaskFilter) ([]*model.Task, error) {
	r, err := s.ResolveBoard(ctx)
	if err != nil {
		return nil, err
	}
	return db.ListTasksInBoard(ctx, s.pool.Write, r.Board.ID, filter)
}

func (s *Service) MoveTask(ctx context.Context, taskID, targetColumnID int64, beforeTaskID *int64) error {
	return db.MoveTaskAt(ctx, s.pool.Write, taskID, targetColumnID, beforeTaskID)
}

func (s *Service) UpdateTask(ctx context.Context, id int64, patch db.TaskPatch) (*model.Task, error) {
	if err := db.UpdateTask(ctx, s.pool.Write, id, patch); err != nil {
		return nil, err
	}
	return db.GetTask(ctx, s.pool.Write, id)
}

func (s *Service) DeleteTask(ctx context.Context, id int64) error {
	return db.DeleteTask(ctx, s.pool.Write, id)
}

func (s *Service) GetTask(ctx context.Context, id int64) (*model.Task, error) {
	return db.GetTask(ctx, s.pool.Write, id)
}

func (s *Service) GetTaskByPrefix(ctx context.Context, prefix string) (*model.Task, error) {
	return db.GetTaskByPrefix(ctx, s.pool.Write, prefix)
}

type ColumnSnapshot struct {
	ID        int64         `json:"id"`
	Name      string        `json:"name"`
	Position  int           `json:"position"`
	WIPLimit  *int          `json:"wip_limit,omitempty"`
	Tasks     []*model.Task `json:"tasks"`
	TaskCount int64         `json:"task_count"`
	Truncated bool          `json:"truncated,omitempty"`
}

type BoardSnapshot struct {
	Project       string           `json:"project"`
	Board         *model.Board     `json:"board"`
	Columns       []ColumnSnapshot `json:"columns"`
	ArchivedCount int64            `json:"archived_count"`
}

type SnapshotOpts struct {
	IncludeDone bool
}

func (s *Service) BoardSnapshot(ctx context.Context) (*BoardSnapshot, error) {
	return s.BoardSnapshotOpts(ctx, SnapshotOpts{})
}

func (s *Service) BoardSnapshotOpts(ctx context.Context, opts SnapshotOpts) (*BoardSnapshot, error) {
	r, err := s.ResolveBoard(ctx)
	if err != nil {
		return nil, err
	}
	cols, err := db.ListColumns(ctx, s.pool.Write, r.Board.ID)
	if err != nil {
		return nil, err
	}
	archivedCount, err := db.CountArchivedInBoard(ctx, s.pool.Write, r.Board.ID)
	if err != nil {
		return nil, err
	}
	out := &BoardSnapshot{
		Project:       r.Project.Name,
		Board:         r.Board,
		Columns:       make([]ColumnSnapshot, 0, len(cols)),
		ArchivedCount: archivedCount,
	}
	for _, c := range cols {
		count, err := db.CountTasksInColumn(ctx, s.pool.Write, c.ID)
		if err != nil {
			return nil, err
		}
		col := ColumnSnapshot{
			ID:        c.ID,
			Name:      c.Name,
			Position:  c.Position,
			WIPLimit:  c.WIPLimit,
			TaskCount: count,
			Tasks:     []*model.Task{},
		}
		terminal := model.IsTerminalColumn(c.Name)
		if terminal && !opts.IncludeDone {
			col.Truncated = count > 0
			out.Columns = append(out.Columns, col)
			continue
		}
		tasks, err := db.ListTasksInColumn(ctx, s.pool.Write, c.ID)
		if err != nil {
			return nil, err
		}
		if tasks == nil {
			tasks = []*model.Task{}
		}
		col.Tasks = tasks
		out.Columns = append(out.Columns, col)
	}
	return out, nil
}

func (s *Service) ArchiveTask(ctx context.Context, id int64) error {
	return db.ArchiveTask(ctx, s.pool.Write, id)
}

func (s *Service) UnarchiveTask(ctx context.Context, id int64) error {
	return db.UnarchiveTask(ctx, s.pool.Write, id)
}

func (s *Service) ArchiveDone(ctx context.Context) (int64, error) {
	r, err := s.ResolveBoard(ctx)
	if err != nil {
		return 0, err
	}
	return db.ArchiveDoneInBoard(ctx, s.pool.Write, r.Board.ID)
}

func (s *Service) ListArchived(ctx context.Context) ([]*model.Task, error) {
	r, err := s.ResolveBoard(ctx)
	if err != nil {
		return nil, err
	}
	return db.ListTasksInBoard(ctx, s.pool.Write, r.Board.ID, db.TaskFilter{OnlyArchived: true})
}

func (s *Service) ApplyTaskTags(ctx context.Context, taskID int64, changes db.TaskTagChanges) error {
	t, err := db.GetTask(ctx, s.pool.Write, taskID)
	if err != nil {
		return err
	}
	board, err := db.GetBoard(ctx, s.pool.Write, t.BoardID)
	if err != nil {
		return err
	}
	return db.ApplyTaskTagChanges(ctx, s.pool.Write, taskID, board.ProjectID, changes)
}

func ParseDue(s string) (*dbutil.Time, error) {
	if s == "" {
		return nil, nil
	}
	t, err := dbutil.ParseDueDate(s)
	if err != nil {
		return nil, err
	}
	return &dbutil.Time{Time: t}, nil
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
