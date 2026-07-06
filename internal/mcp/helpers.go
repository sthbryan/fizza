package mcp

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/fizza/fizza/internal/db"
	"github.com/fizza/fizza/internal/dbutil"
	"github.com/fizza/fizza/internal/model"
	"github.com/jmoiron/sqlx"
)

func splitCSV(s string) []string {
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

func parsePriority(s string) (model.Priority, error) {
	if s == "" {
		return model.Priority{Value: model.DefaultPriority}, nil
	}
	return model.NewPriority(s)
}

func findBoardAndColumns(ctx context.Context, conn *sqlx.DB, project, board string) (*model.Board, []*model.Column, error) {
	p, err := db.GetProjectByName(ctx, conn, project)
	if err != nil {
		return nil, nil, err
	}
	boards, err := db.ListBoards(ctx, conn, p.ID)
	if err != nil {
		return nil, nil, err
	}
	var found *model.Board
	for _, b := range boards {
		if b.Name == board {
			found = b
			break
		}
	}
	if found == nil {
		return nil, nil, fmt.Errorf("%w: board %q in project %q", db.ErrNotFound, board, project)
	}
	cols, err := db.ListColumns(ctx, conn, found.ID)
	if err != nil {
		return nil, nil, err
	}
	return found, cols, nil
}

func pickColumn(cols []*model.Column, name string) (*model.Column, error) {
	if name == "" {
		if len(cols) == 0 {
			return nil, fmt.Errorf("%w: board has no columns", db.ErrNotFound)
		}
		return cols[0], nil
	}
	for _, c := range cols {
		if c.Name == name {
			return c, nil
		}
	}
	return nil, fmt.Errorf("%w: column %q", db.ErrNotFound, name)
}

func parseInt64Flexible(s string) (int64, error) {
	if s == "" {
		return 0, fmt.Errorf("empty")
	}
	var n int64
	for _, r := range s {
		if r < '0' || r > '9' {
			return 0, fmt.Errorf("not numeric: %q", s)
		}
		n = n*10 + int64(r-'0')
	}
	return n, nil
}

func parseDue(s string) (*time.Time, error) {
	if s == "" {
		return nil, nil
	}
	t, err := dbutil.ParseDueDate(s)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func buildTaskFromInput(board *model.Board, target *model.Column, title string, priority model.Priority, desc, due, parent string) (*model.Task, error) {
	t := &model.Task{
		BoardID:     board.ID,
		ColumnID:    target.ID,
		Title:       title,
		Description: desc,
		Priority:    priority,
	}
	parsed, err := parseDue(due)
	if err != nil {
		return nil, fmt.Errorf("due: %w", err)
	}
	t.DueDate = parsed
	if parent != "" {
		pid, err := parseInt64Flexible(parent)
		if err != nil {
			return nil, fmt.Errorf("parent: %w", err)
		}
		t.ParentID = &pid
	}
	return t, nil
}

func buildTaskPatch(in taskUpdateInput) (db.TaskPatch, error) {
	patch := db.TaskPatch{}
	if in.Title != "" {
		v := in.Title
		patch.Title = &v
	}
	if in.Desc != "" {
		v := in.Desc
		patch.Description = &v
	}
	if in.Priority != "" {
		pri, err := model.NewPriority(in.Priority)
		if err != nil {
			return patch, err
		}
		patch.Priority = &pri
	}
	if in.ClearDue {
		patch.ClearDueDate = true
	} else if in.Due != "" {
		parsed, err := dbutil.ParseDueDate(in.Due)
		if err != nil {
			return patch, fmt.Errorf("due: %w", err)
		}
		patch.DueDate = &parsed
	}
	if in.ClearParent {
		patch.ClearParentID = true
	} else if in.Parent != "" {
		pid, err := parseInt64Flexible(in.Parent)
		if err != nil {
			return patch, fmt.Errorf("parent: %w", err)
		}
		patch.ParentID = &pid
	}
	return patch, nil
}