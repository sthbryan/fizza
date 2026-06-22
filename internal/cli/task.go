package cli

import (
	"context"
	"database/sql"
	"fmt"
	"os"

	"github.com/fizza/fizza/internal/db"
	"github.com/fizza/fizza/internal/dbutil"
	"github.com/fizza/fizza/internal/model"
	"github.com/spf13/cobra"
)

func newTaskCmd(rf *rootFlags) *cobra.Command {
	cmd := &cobra.Command{Use: "task", Short: "Manage tasks"}
	cmd.AddCommand(newTaskAddCmd(rf))
	cmd.AddCommand(newTaskListCmd(rf))
	cmd.AddCommand(newTaskShowCmd(rf))
	cmd.AddCommand(newTaskMoveCmd(rf))
	cmd.AddCommand(newTaskUpdateCmd(rf))
	cmd.AddCommand(newTaskDeleteCmd(rf))
	return cmd
}

func newTaskAddCmd(rf *rootFlags) *cobra.Command {
	var project, board, column, desc, priority, due, parent string
	c := &cobra.Command{
		Use:   "add <title>",
		Short: "Add a task to a board",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := mustArgs(cmd, args, 1); err != nil {
				return report(cmd, rf, err)
			}
			if err := mustFlags(cmd, "project", "board"); err != nil {
				return report(cmd, rf, err)
			}
			ctx := cmd.Context()
			conn, err := rf.openDB(ctx)
			if err != nil {
				return report(cmd, rf, err)
			}
			defer conn.Close()

			b, err := findBoard(ctx, conn, project, board)
			if err != nil {
				return report(cmd, rf, err)
			}
			targetCol, err := resolveColumn(ctx, conn, b.ID, column)
			if err != nil {
				return report(cmd, rf, err)
			}
			pri, err := model.ParsePriority(priority)
			if err != nil {
				return report(cmd, rf, err)
			}
			t := &model.Task{
				BoardID:     b.ID,
				ColumnID:    targetCol.ID,
				Title:       args[0],
				Description: desc,
				Priority:    pri,
			}
			if due != "" {
				parsed, err := dbutil.ParseDueDate(due)
				if err != nil {
					return report(cmd, rf, err)
				}
				t.DueDate = &parsed
			}
			if parent != "" {
				pid, err := parseInt64(parent)
				if err != nil {
					return report(cmd, rf, fmt.Errorf("invalid --parent: %w", err))
				}
				t.ParentID = &pid
			}
			if err := db.CreateTask(ctx, conn, t); err != nil {
				return report(cmd, rf, err)
			}
			return writeOK(cmd, rf, t)
		},
	}
	c.Flags().StringVar(&project, "project", "", "Project name (required)")
	c.Flags().StringVar(&board, "board", "", "Board name (required)")
	c.Flags().StringVar(&column, "column", "", "Column name (default: first column)")
	c.Flags().StringVar(&desc, "desc", "", "Task description")
	c.Flags().StringVar(&priority, "priority", model.DefaultPriority, "low|medium|high|urgent")
	c.Flags().StringVar(&due, "due", "", "Due date (YYYY-MM-DD or ISO 8601)")
	c.Flags().StringVar(&parent, "parent", "", "Parent task ID (numeric prefix)")
	return c
}

func newTaskListCmd(rf *rootFlags) *cobra.Command {
	var project, board, column string
	c := &cobra.Command{
		Use:   "list",
		Short: "List tasks in a board",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := mustFlags(cmd, "project", "board"); err != nil {
				return report(cmd, rf, err)
			}
			ctx := cmd.Context()
			conn, err := rf.openDB(ctx)
			if err != nil {
				return report(cmd, rf, err)
			}
			defer conn.Close()

			b, err := findBoard(ctx, conn, project, board)
			if err != nil {
				return report(cmd, rf, err)
			}
			tasks, err := db.ListTasksInBoard(ctx, conn, b.ID, column)
			if err != nil {
				return report(cmd, rf, err)
			}
			if tasks == nil {
				tasks = []*model.Task{}
			}
			return writeOK(cmd, rf, tasks)
		},
	}
	c.Flags().StringVar(&project, "project", "", "Project name (required)")
	c.Flags().StringVar(&board, "board", "", "Board name (required)")
	c.Flags().StringVar(&column, "column", "", "Filter by column name")
	return c
}

func newTaskShowCmd(rf *rootFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "show <id>",
		Short: "Show a task by ID or numeric prefix",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := mustArgs(cmd, args, 1); err != nil {
				return report(cmd, rf, err)
			}
			ctx := cmd.Context()
			conn, err := rf.openDB(ctx)
			if err != nil {
				return report(cmd, rf, err)
			}
			defer conn.Close()

			t, err := db.GetTaskByPrefix(ctx, conn, args[0])
			if err != nil {
				return report(cmd, rf, err)
			}
			return writeOK(cmd, rf, t)
		},
	}
}

func newTaskMoveCmd(rf *rootFlags) *cobra.Command {
	var project, board string
	c := &cobra.Command{
		Use:   "move <id> <column>",
		Short: "Move a task to a column",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := mustArgs(cmd, args, 2); err != nil {
				return report(cmd, rf, err)
			}
			if err := mustFlags(cmd, "project", "board"); err != nil {
				return report(cmd, rf, err)
			}
			ctx := cmd.Context()
			conn, err := rf.openDB(ctx)
			if err != nil {
				return report(cmd, rf, err)
			}
			defer conn.Close()

			t, err := db.GetTaskByPrefix(ctx, conn, args[0])
			if err != nil {
				return report(cmd, rf, err)
			}
			b, err := findBoard(ctx, conn, project, board)
			if err != nil {
				return report(cmd, rf, err)
			}
			if t.BoardID != b.ID {
				return report(cmd, rf, fmt.Errorf("%w: task belongs to a different board", db.ErrNotFound))
			}
			target, err := db.GetColumnByName(ctx, conn, b.ID, args[1])
			if err != nil {
				return report(cmd, rf, err)
			}
			if err := db.MoveTask(ctx, conn, t.ID, target.ID); err != nil {
				return report(cmd, rf, err)
			}
			updated, err := db.GetTask(ctx, conn, t.ID)
			if err != nil {
				return report(cmd, rf, err)
			}
			return writeOK(cmd, rf, updated)
		},
	}
	c.Flags().StringVar(&project, "project", "", "Project name (required)")
	c.Flags().StringVar(&board, "board", "", "Board name (required)")
	return c
}

func newTaskUpdateCmd(rf *rootFlags) *cobra.Command {
	var title, desc, priority, due, parent string
	var clearDue, clearParent bool
	c := &cobra.Command{
		Use:   "update <id>",
		Short: "Update a task's fields",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := mustArgs(cmd, args, 1); err != nil {
				return report(cmd, rf, err)
			}
			ctx := cmd.Context()
			conn, err := rf.openDB(ctx)
			if err != nil {
				return report(cmd, rf, err)
			}
			defer conn.Close()

			t, err := db.GetTaskByPrefix(ctx, conn, args[0])
			if err != nil {
				return report(cmd, rf, err)
			}

			patch := db.TaskPatch{}
			if cmd.Flags().Changed("title") {
				patch.Title = &title
			}
			if cmd.Flags().Changed("desc") {
				patch.Description = &desc
			}
			if cmd.Flags().Changed("priority") {
				pri, err := model.ParsePriority(priority)
				if err != nil {
					return report(cmd, rf, err)
				}
				patch.Priority = &pri
			}
			if clearDue {
				patch.ClearDueDate = true
			} else if due != "" {
				parsed, err := dbutil.ParseDueDate(due)
				if err != nil {
					return report(cmd, rf, err)
				}
				patch.DueDate = &parsed
			}
			if clearParent {
				patch.ClearParentID = true
			} else if parent != "" {
				pid, err := parseInt64(parent)
				if err != nil {
					return report(cmd, rf, fmt.Errorf("invalid --parent: %w", err))
				}
				patch.ParentID = &pid
			}

			if err := db.UpdateTask(ctx, conn, t.ID, patch); err != nil {
				return report(cmd, rf, err)
			}
			updated, err := db.GetTask(ctx, conn, t.ID)
			if err != nil {
				return report(cmd, rf, err)
			}
			return writeOK(cmd, rf, updated)
		},
	}
	c.Flags().StringVar(&title, "title", "", "New title")
	c.Flags().StringVar(&desc, "desc", "", "New description")
	c.Flags().StringVar(&priority, "priority", "", "New priority")
	c.Flags().StringVar(&due, "due", "", "New due date")
	c.Flags().BoolVar(&clearDue, "clear-due", false, "Remove due date")
	c.Flags().StringVar(&parent, "parent", "", "New parent task ID")
	c.Flags().BoolVar(&clearParent, "clear-parent", false, "Remove parent")
	return c
}

func newTaskDeleteCmd(rf *rootFlags) *cobra.Command {
	var force bool
	c := &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a task (cascades subtasks)",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := mustArgs(cmd, args, 1); err != nil {
				return report(cmd, rf, err)
			}
			ctx := cmd.Context()
			conn, err := rf.openDB(ctx)
			if err != nil {
				return report(cmd, rf, err)
			}
			defer conn.Close()

			t, err := db.GetTaskByPrefix(ctx, conn, args[0])
			if err != nil {
				return report(cmd, rf, err)
			}
			if !force {
				env := Fail(CodeConflict, fmt.Sprintf("refusing to delete task %d without --force", t.ID))
				out := rf.output(cmd.ErrOrStderr())
				_ = out.Write(env)
				os.Exit(ExitConflict)
			}
			if err := db.DeleteTask(ctx, conn, t.ID); err != nil {
				return report(cmd, rf, err)
			}
			return writeOK(cmd, rf, map[string]any{"deleted": t.ID, "title": t.Title})
		},
	}
	c.Flags().BoolVar(&force, "force", false, "Skip confirmation")
	return c
}

func findBoard(ctx context.Context, conn *sql.DB, project, board string) (*model.Board, error) {
	p, err := db.GetProjectByName(ctx, conn, project)
	if err != nil {
		return nil, err
	}
	boards, err := db.ListBoards(ctx, conn, p.ID)
	if err != nil {
		return nil, err
	}
	for _, b := range boards {
		if b.Name == board {
			return b, nil
		}
	}
	return nil, fmt.Errorf("%w: board %q in project %q", db.ErrNotFound, board, project)
}

func resolveColumn(ctx context.Context, conn *sql.DB, boardID int64, name string) (*model.Column, error) {
	cols, err := db.ListColumns(ctx, conn, boardID)
	if err != nil {
		return nil, err
	}
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

func parseInt64(s string) (int64, error) {
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