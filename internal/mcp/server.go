package mcp

import (
	"context"
	"fmt"
	"os"

	"github.com/fizza/fizza/internal/config"
	"github.com/fizza/fizza/internal/db"
	"github.com/fizza/fizza/internal/model"
	"github.com/jmoiron/sqlx"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const serverName = "fizza"

func Run(ctx context.Context, version string) error {
	path, err := config.DBPath()
	if err != nil {
		return fmt.Errorf("resolve db path: %w", err)
	}
	pool, err := db.OpenPool(ctx, path, 4, 1)
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}
	defer pool.Close()
	conn := pool.Write

	if version == "" {
		version = "dev"
	}

	server := mcp.NewServer(&mcp.Implementation{
		Name:    serverName,
		Version: version,
	}, &mcp.ServerOptions{
		Instructions: "Fizza is a kanban board manager backed by SQLite. Use these tools to manage projects, boards, and tasks.",
	})

	registerProjectTools(server, conn)
	registerBoardTools(server, conn)
	registerTaskTools(server, conn)
	registerTagTools(server, conn)

	if err := server.Run(ctx, &mcp.StdioTransport{}); err != nil {
		fmt.Fprintln(os.Stderr, "mcp server:", err)
		return err
	}
	return nil
}

type projectInput struct {
	Name        string `json:"name" jsonschema:"project name (required, must be unique)"`
	Description string `json:"description,omitempty" jsonschema:"optional project description"`
}

type projectListInput struct {
	Name string `json:"name,omitempty" jsonschema:"project name; if set, returns single project instead of list"`
}

type projectDeleteInput struct {
	Name  string `json:"name" jsonschema:"project name"`
	Force bool   `json:"force,omitempty" jsonschema:"skip confirmation"`
}

func registerProjectTools(s *mcp.Server, conn *sqlx.DB) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "project_new",
		Description: "Create a new project. A default board named 'main' with columns todo/in_progress/done is seeded automatically.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, in projectInput) (*mcp.CallToolResult, any, error) {
		p, err := db.CreateProject(ctx, conn, in.Name, in.Description)
		if err != nil {
			return nil, nil, err
		}
		return nil, p, nil
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "project_list",
		Description: "List projects, or fetch one by name. Omit name to list all.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, in projectListInput) (*mcp.CallToolResult, any, error) {
		if in.Name != "" {
			p, err := db.GetProjectByName(ctx, conn, in.Name)
			if err != nil {
				return nil, nil, err
			}
			return nil, p, nil
		}
		out, err := db.ListProjects(ctx, conn)
		if err != nil {
			return nil, nil, err
		}
		if out == nil {
			return nil, []*model.Project{}, nil
		}
		return nil, out, nil
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "project_delete",
		Description: "Delete a project. Cascades to boards, columns, and tasks. Requires force=true to actually delete.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, in projectDeleteInput) (*mcp.CallToolResult, any, error) {
		p, err := db.GetProjectByName(ctx, conn, in.Name)
		if err != nil {
			return nil, nil, err
		}
		if !in.Force {
			return nil, nil, fmt.Errorf("refusing to delete %q: pass force=true to confirm", in.Name)
		}
		if err := db.DeleteProject(ctx, conn, p.ID); err != nil {
			return nil, nil, err
		}
		return nil, map[string]any{"deleted": p.Name, "id": p.ID}, nil
	})
}

type boardCreateInput struct {
	Project string `json:"project" jsonschema:"project name (required)"`
	Name    string `json:"name" jsonschema:"board name (required)"`
	Columns string `json:"columns,omitempty" jsonschema:"comma-separated column names; defaults to todo,in_progress,done"`
}

type boardInProjectInput struct {
	Project string `json:"project" jsonschema:"project name"`
	Name    string `json:"name" jsonschema:"board name"`
}

type boardListInput struct {
	Project string `json:"project" jsonschema:"project name"`
	Name    string `json:"name,omitempty" jsonschema:"board name; if set, returns single-element array"`
}

type boardDeleteInput struct {
	Project string `json:"project" jsonschema:"project name"`
	Name    string `json:"name" jsonschema:"board name"`
	Force   bool   `json:"force,omitempty" jsonschema:"skip confirmation"`
}

func registerBoardTools(s *mcp.Server, conn *sqlx.DB) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "board_create",
		Description: "Create a board in a project. Optionally seed it with custom column names.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, in boardCreateInput) (*mcp.CallToolResult, any, error) {
		p, err := db.GetProjectByName(ctx, conn, in.Project)
		if err != nil {
			return nil, nil, err
		}
		b, err := db.CreateBoardWithColumns(ctx, conn, p.ID, in.Name, splitCSV(in.Columns))
		if err != nil {
			return nil, nil, err
		}
		return nil, b, nil
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "board_list",
		Description: "List boards in a project, or fetch one by name. Omit name to list all.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, in boardListInput) (*mcp.CallToolResult, any, error) {
		p, err := db.GetProjectByName(ctx, conn, in.Project)
		if err != nil {
			return nil, nil, err
		}
		if in.Name != "" {
			board, _, err := findBoardAndColumns(ctx, conn, in.Project, in.Name)
			if err != nil {
				return nil, nil, err
			}
			return nil, []*model.Board{board}, nil
		}
		boards, err := db.ListBoards(ctx, conn, p.ID)
		if err != nil {
			return nil, nil, err
		}
		if boards == nil {
			return nil, []*model.Board{}, nil
		}
		return nil, boards, nil
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "board_delete",
		Description: "Delete a board. Blocked by RESTRICT FK if it contains tasks; pass force=true to confirm intent (the FK still blocks).",
	}, func(ctx context.Context, req *mcp.CallToolRequest, in boardDeleteInput) (*mcp.CallToolResult, any, error) {
		board, _, err := findBoardAndColumns(ctx, conn, in.Project, in.Name)
		if err != nil {
			return nil, nil, err
		}
		if !in.Force {
			return nil, nil, fmt.Errorf("refusing to delete board %q: pass force=true to confirm", in.Name)
		}
		if err := db.DeleteBoard(ctx, conn, board.ID); err != nil {
			return nil, nil, err
		}
		return nil, map[string]any{"deleted": board.Name, "id": board.ID}, nil
	})
}

type taskAddInput struct {
	Project  string `json:"project" jsonschema:"project name (required)"`
	Board    string `json:"board" jsonschema:"board name (required)"`
	Title    string `json:"title" jsonschema:"task title (required)"`
	Column   string `json:"column,omitempty" jsonschema:"column name; defaults to first column"`
	Desc     string `json:"desc,omitempty" jsonschema:"task description"`
	Priority string `json:"priority,omitempty" jsonschema:"low|medium|high|urgent (default medium)"`
	Due      string `json:"due,omitempty" jsonschema:"due date (YYYY-MM-DD or ISO 8601)"`
	Parent   string `json:"parent,omitempty" jsonschema:"parent task ID or numeric prefix"`
}

type taskListInput struct {
	Project  string `json:"project,omitempty" jsonschema:"project name (defaults to configured project)"`
	Board    string `json:"board,omitempty" jsonschema:"board name (defaults to configured board)"`
	ID       string `json:"id,omitempty" jsonschema:"task ID or numeric prefix; if set, returns single-element array"`
	Column   string `json:"column,omitempty" jsonschema:"filter by column name (acts as status filter)"`
	Priority string `json:"priority,omitempty" jsonschema:"filter by priority: low|medium|high|urgent"`
	Tag      string `json:"tag,omitempty" jsonschema:"filter by tag name"`
	Search   string `json:"search,omitempty" jsonschema:"substring match against title/description"`
}

type taskMoveInput struct {
	ID      string `json:"id" jsonschema:"task ID or numeric prefix"`
	Project string `json:"project" jsonschema:"project name (required)"`
	Board   string `json:"board" jsonschema:"board name (required)"`
	Column  string `json:"column" jsonschema:"target column name (required)"`
}

type taskUpdateInput struct {
	ID          string   `json:"id" jsonschema:"task ID or numeric prefix"`
	Title       string   `json:"title,omitempty" jsonschema:"new title"`
	Desc        string   `json:"desc,omitempty" jsonschema:"new description"`
	Priority    string   `json:"priority,omitempty" jsonschema:"new priority"`
	Due         string   `json:"due,omitempty" jsonschema:"new due date"`
	ClearDue    bool     `json:"clear_due,omitempty" jsonschema:"remove the due date"`
	Parent      string   `json:"parent,omitempty" jsonschema:"new parent task ID"`
	ClearParent bool     `json:"clear_parent,omitempty" jsonschema:"remove the parent"`
	AddTags     []string `json:"add_tags,omitempty" jsonschema:"tag names to attach; missing tags are auto-created"`
	RemoveTags  []string `json:"remove_tags,omitempty" jsonschema:"tag names to detach"`
}

type taskDeleteInput struct {
	ID    string `json:"id" jsonschema:"task ID or numeric prefix"`
	Force bool   `json:"force,omitempty" jsonschema:"skip confirmation"`
}

func registerTaskTools(s *mcp.Server, conn *sqlx.DB) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "task_add",
		Description: "Create a task in a board. By default lands in the first column. Use task_move to change column afterward.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, in taskAddInput) (*mcp.CallToolResult, any, error) {
		board, cols, err := findBoardAndColumns(ctx, conn, in.Project, in.Board)
		if err != nil {
			return nil, nil, err
		}
		target, err := pickColumn(cols, in.Column)
		if err != nil {
			return nil, nil, err
		}
		pri, err := parsePriority(in.Priority)
		if err != nil {
			return nil, nil, err
		}
		t, err := buildTaskFromInput(board, target, in.Title, pri, in.Desc, in.Due, in.Parent)
		if err != nil {
			return nil, nil, err
		}
		if err := db.CreateTask(ctx, conn, t); err != nil {
			return nil, nil, err
		}
		return nil, t, nil
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "task_list",
		Description: "List tasks in a board with optional filters (column, priority, tag, search), or fetch one by id.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, in taskListInput) (*mcp.CallToolResult, any, error) {
		if in.ID != "" {
			t, err := db.GetTaskByPrefix(ctx, conn, in.ID)
			if err != nil {
				return nil, nil, err
			}
			return nil, []*model.Task{t}, nil
		}
		board, _, err := findBoardAndColumns(ctx, conn, in.Project, in.Board)
		if err != nil {
			return nil, nil, err
		}
		filter := db.TaskFilter{ColumnName: in.Column, Search: in.Search}
		if in.Priority != "" {
			pri, err := model.NewPriority(in.Priority)
			if err != nil {
				return nil, nil, fmt.Errorf("priority: %w", err)
			}
			filter.Priorities = []model.Priority{pri}
		}
		if in.Tag != "" {
			filter.Tags = []string{in.Tag}
		}
		tasks, err := db.ListTasksInBoard(ctx, conn, board.ID, filter)
		if err != nil {
			return nil, nil, err
		}
		if tasks == nil {
			return nil, []*model.Task{}, nil
		}
		return nil, tasks, nil
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "task_move",
		Description: "Move a task to a different column within its board. Goes to the bottom of the target column.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, in taskMoveInput) (*mcp.CallToolResult, any, error) {
		t, err := db.GetTaskByPrefix(ctx, conn, in.ID)
		if err != nil {
			return nil, nil, err
		}
		board, _, err := findBoardAndColumns(ctx, conn, in.Project, in.Board)
		if err != nil {
			return nil, nil, err
		}
		if t.BoardID != board.ID {
			return nil, nil, fmt.Errorf("task %d belongs to a different board", t.ID)
		}
		target, err := db.GetColumnByName(ctx, conn, board.ID, in.Column)
		if err != nil {
			return nil, nil, err
		}
		if err := db.MoveTask(ctx, conn, t.ID, target.ID); err != nil {
			return nil, nil, err
		}
		updated, err := db.GetTask(ctx, conn, t.ID)
		if err != nil {
			return nil, nil, err
		}
		return nil, updated, nil
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "task_update",
		Description: "Update one or more fields of a task. add_tags/remove_tags attach/detach tags in the same call.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, in taskUpdateInput) (*mcp.CallToolResult, any, error) {
		t, err := db.GetTaskByPrefix(ctx, conn, in.ID)
		if err != nil {
			return nil, nil, err
		}
		patch, err := buildTaskPatch(in)
		if err != nil {
			return nil, nil, err
		}
		if err := db.UpdateTask(ctx, conn, t.ID, patch); err != nil {
			return nil, nil, err
		}
		if len(in.AddTags) > 0 || len(in.RemoveTags) > 0 {
			board, err := db.GetBoard(ctx, conn, t.BoardID)
			if err != nil {
				return nil, nil, err
			}
			if err := db.ApplyTaskTagChanges(ctx, conn, t.ID, board.ProjectID, db.TaskTagChanges{
				Add:    in.AddTags,
				Remove: in.RemoveTags,
			}); err != nil {
				return nil, nil, err
			}
		}
		updated, err := db.GetTask(ctx, conn, t.ID)
		if err != nil {
			return nil, nil, err
		}
		return nil, updated, nil
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "task_delete",
		Description: "Delete a task and its subtasks (cascade). Requires force=true.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, in taskDeleteInput) (*mcp.CallToolResult, any, error) {
		t, err := db.GetTaskByPrefix(ctx, conn, in.ID)
		if err != nil {
			return nil, nil, err
		}
		if !in.Force {
			return nil, nil, fmt.Errorf("refusing to delete task %d: pass force=true", t.ID)
		}
		if err := db.DeleteTask(ctx, conn, t.ID); err != nil {
			return nil, nil, err
		}
		return nil, map[string]any{"deleted": t.ID, "title": t.Title}, nil
	})
}

type tagInput struct {
	Project string `json:"project,omitempty" jsonschema:"project name (defaults to current)"`
	Name    string `json:"name" jsonschema:"tag name"`
}

type tagListInput struct {
	Project string `json:"project,omitempty" jsonschema:"project name"`
}

type tagDeleteInput struct {
	ID    int64 `json:"id" jsonschema:"tag id"`
	Force bool  `json:"force,omitempty" jsonschema:"skip confirmation"`
}

func registerTagTools(s *mcp.Server, conn *sqlx.DB) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "tag_add",
		Description: "Create a tag in a project.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, in tagInput) (*mcp.CallToolResult, any, error) {
		project := in.Project
		if project == "" {
			p, err := defaultProject()
			if err != nil {
				return nil, nil, err
			}
			project = p
		}
		p, err := db.GetProjectByName(ctx, conn, project)
		if err != nil {
			return nil, nil, err
		}
		t, err := db.CreateTag(ctx, conn, p.ID, in.Name)
		if err != nil {
			return nil, nil, err
		}
		return nil, t, nil
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "tag_list",
		Description: "List tags in a project.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, in tagListInput) (*mcp.CallToolResult, any, error) {
		project := in.Project
		if project == "" {
			project, _ = defaultProject()
		}
		if project == "" {
			return nil, nil, fmt.Errorf("project required")
		}
		p, err := db.GetProjectByName(ctx, conn, project)
		if err != nil {
			return nil, nil, err
		}
		tags, err := db.ListTags(ctx, conn, p.ID)
		if err != nil {
			return nil, nil, err
		}
		if tags == nil {
			tags = []*model.Tag{}
		}
		return nil, tags, nil
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "tag_delete",
		Description: "Delete a tag. Requires force=true.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, in tagDeleteInput) (*mcp.CallToolResult, any, error) {
		if !in.Force {
			return nil, nil, fmt.Errorf("refusing to delete tag %d: pass force=true", in.ID)
		}
		if err := db.DeleteTag(ctx, conn, in.ID); err != nil {
			return nil, nil, err
		}
		return nil, map[string]any{"deleted": in.ID}, nil
	})

}

func defaultProject() (string, error) {
	cfg, err := config.LoadConfig()
	if err != nil {
		return "", err
	}
	if cfg.Project == "" {
		return "", fmt.Errorf("no default project set")
	}
	return cfg.Project, nil
}
