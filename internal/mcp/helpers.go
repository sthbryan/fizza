package mcp

import (
	"context"
	"fmt"
	"os"

	"github.com/fizza/fizza/internal/config"
	"github.com/fizza/fizza/internal/db"
	"github.com/fizza/fizza/internal/dbutil"
	"github.com/fizza/fizza/internal/model"
	"github.com/fizza/fizza/internal/service"
	"github.com/jmoiron/sqlx"
)

func splitCSV(s string) []string {
	return service.SplitColumns(s)
}

func parsePriority(s string) (model.Priority, error) {
	if s == "" {
		return model.Priority{Value: model.DefaultPriority}, nil
	}
	return model.NewPriority(s)
}

func parseInt64Flexible(s string) (int64, error) {
	return service.ParseInt64Flexible(s)
}

func parseDue(s string) (*dbutil.Time, error) {
	return service.ParseDue(s)
}

func loadDefaults() (project, board string) {
	cwd, err := os.Getwd()
	if err != nil {
		cwd = ""
	}
	cfg, err := config.LoadEffectiveConfig(cwd)
	if err != nil {
		return "", ""
	}
	return cfg.Project, cfg.Board
}

func resolveScope(project, board string) (string, string, error) {
	defProj, defBoard := loadDefaults()
	if project == "" {
		project = defProj
	}
	if board == "" {
		board = defBoard
	}
	if project == "" {
		return "", "", fmt.Errorf("%w: project required (pass project or run `fizza config set project <name>`)", model.ErrValidation)
	}
	return project, board, nil
}

func newService(pool *db.Pool, project, board, column string) *service.Service {
	return service.NewWithPool(pool, project, board, column)
}

func findBoardAndColumns(ctx context.Context, conn *sqlx.DB, project, board string) (*model.Board, []*model.Column, error) {
	project, board, err := resolveScope(project, board)
	if err != nil {
		return nil, nil, err
	}
	if board == "" {
		p, err := db.GetProjectByName(ctx, conn, project)
		if err != nil {
			return nil, nil, err
		}
		boards, err := db.ListBoards(ctx, conn, p.ID)
		if err != nil {
			return nil, nil, err
		}
		if len(boards) == 0 {
			return nil, nil, fmt.Errorf("%w: project %q has no boards", db.ErrNotFound, project)
		}
		if len(boards) > 1 {
			return nil, nil, fmt.Errorf("%w: project %q has %d boards; pass board or set a default", model.ErrValidation, project, len(boards))
		}
		board = boards[0].Name
	}
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
		patch.DueDate = &dbutil.Time{Time: parsed}
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

func resolveProjectName(project string) (string, error) {
	if project != "" {
		return project, nil
	}
	p, _ := loadDefaults()
	if p == "" {
		return "", fmt.Errorf("%w: project required (pass project or run `fizza config set project <name>`)", model.ErrValidation)
	}
	return p, nil
}
