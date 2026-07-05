package cli

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/fizza/fizza/internal/db"
	"github.com/fizza/fizza/internal/model"
	"github.com/fizza/fizza/internal/service"
	"github.com/spf13/cobra"
)

func newTaskCmd(rf *rootFlags) *cobra.Command {
	cmd := &cobra.Command{Use: "task", Short: "Manage tasks"}
	cmd.AddCommand(newTaskAddCmd(rf))
	cmd.AddCommand(newTaskBulkCmd(rf))
	cmd.AddCommand(newTaskListCmd(rf))
	cmd.AddCommand(newTaskShowCmd(rf))
	cmd.AddCommand(newTaskTreeCmd(rf))
	cmd.AddCommand(newTaskMoveCmd(rf))
	cmd.AddCommand(newTaskUpdateCmd(rf))
	cmd.AddCommand(newTaskDeleteCmd(rf))
	cmd.AddCommand(newTaskHistoryCmd(rf))
	return cmd
}

func newTaskAddCmd(rf *rootFlags) *cobra.Command {
	var board, column, desc, priority, due, parent string
	c := &cobra.Command{
		Use:   "add <title>",
		Short: "Add a task to a board",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := mustArgs(cmd, args, 1); err != nil {
				return report(cmd, rf, err)
			}
			if err := mustFlags(cmd, "board"); err != nil {
				return report(cmd, rf, err)
			}
			ctx := cmd.Context()
			svc, err := rf.openDBWith(ctx, "", board, column)
			if err != nil {
				return report(cmd, rf, err)
			}
			defer svc.Close()

			if _, err := svc.ResolveBoard(ctx); err != nil {
				return report(cmd, rf, err)
			}

			pri, err := model.NewPriority(priority)
			if err != nil {
				return report(cmd, rf, err)
			}
			in := serviceTaskInput(args[0], desc, pri, due, parent)
			t, err := svc.CreateTask(ctx, in)
			if err != nil {
				return report(cmd, rf, err)
			}
			return writeOK(cmd, rf, t)
		},
	}
	c.Flags().StringVar(&board, "board", "", "Board name (required)")
	c.Flags().StringVar(&column, "column", "", "Column name (default: first column)")
	c.Flags().StringVar(&desc, "desc", "", "Task description")
	c.Flags().StringVar(&priority, "priority", model.DefaultPriority, "low|medium|high|urgent")
	c.Flags().StringVar(&due, "due", "", "Due date (YYYY-MM-DD or ISO 8601)")
	c.Flags().StringVar(&parent, "parent", "", "Parent task ID (numeric prefix)")
	return c
}

func serviceTaskInput(title, desc string, pri model.Priority, due, parent string) taskInput {
	in := taskInput{Title: title, Description: desc, Priority: pri}
	if due != "" {
		parsed, err := parseCLIDueDate(due)
		if err == nil {
			in.DueDate = &parsed
		}
	}
	if parent != "" {
		pid, err := parseInt64(parent)
		if err == nil {
			in.ParentID = &pid
		}
	}
	return in
}

type taskInput = struct {
	Title       string
	Description string
	Priority    model.Priority
	DueDate     *time.Time
	ParentID    *int64
	ColumnID    int64
}

func parseCLIDueDate(s string) (time.Time, error) {
	for _, layout := range []string{"2006-01-02", time.RFC3339Nano, time.RFC3339} {
		if t, err := time.Parse(layout, s); err == nil {
			return t.UTC(), nil
		}
	}
	return time.Time{}, fmt.Errorf("unparseable date %q", s)
}

func newTaskBulkCmd(rf *rootFlags) *cobra.Command {
	var board, fromFile string
	c := &cobra.Command{
		Use:   "bulk add",
		Short: "Add many tasks from a JSON file",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := mustFlags(cmd, "board", "from"); err != nil {
				return report(cmd, rf, err)
			}
			data, err := readFileOrStdin(fromFile, cmd.InOrStdin())
			if err != nil {
				return report(cmd, rf, err)
			}
			var items []bulkTaskSpec
			if err := decodeJSON(data, &items); err != nil {
				return report(cmd, rf, fmt.Errorf("%w: invalid JSON: %v", ErrValidation, err))
			}
			ctx := cmd.Context()
			svc, err := rf.openDBWith(ctx, "", board, "")
			if err != nil {
				return report(cmd, rf, err)
			}
			defer svc.Close()
			r, err := svc.ResolveBoard(ctx)
			if err != nil {
				return report(cmd, rf, err)
			}
			created := []*model.Task{}
			for i, item := range items {
				pri, err := model.NewPriority(item.Priority)
				if err != nil {
					return report(cmd, rf, fmt.Errorf("%w: item %d priority: %v", ErrValidation, i, err))
				}
				in := taskInput{
					Title:       strings.TrimSpace(item.Title),
					Description: item.Description,
					Priority:    pri,
				}
				if item.Due != "" {
					d, err := parseCLIDueDate(item.Due)
					if err != nil {
						return report(cmd, rf, fmt.Errorf("%w: item %d due: %v", ErrValidation, i, err))
					}
					in.DueDate = &d
				}
				if item.Column != "" {
					c, err := db.GetColumnByName(ctx, svc.DB(), r.Board.ID, item.Column)
					if err != nil {
						return report(cmd, rf, fmt.Errorf("%w: item %d column: %v", ErrValidation, i, err))
					}
					in.ColumnID = c.ID
				}
				t, err := svc.CreateTask(ctx, in)
				if err != nil {
					return report(cmd, rf, fmt.Errorf("%w: item %d: %v", ErrValidation, i, err))
				}
				created = append(created, t)
			}
			return writeOK(cmd, rf, map[string]any{
				"created": len(created),
				"tasks":   created,
			})
		},
	}
	c.Flags().StringVar(&board, "board", "", "Board name (required)")
	c.Flags().StringVar(&fromFile, "from", "", "Path to JSON file (or - for stdin)")
	return c
}

type bulkTaskSpec struct {
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	Priority    string `json:"priority,omitempty"`
	Due         string `json:"due,omitempty"`
	Column      string `json:"column,omitempty"`
}

func newTaskListCmd(rf *rootFlags) *cobra.Command {
	var board, column, priority, dueBefore, dueAfter, search string
	c := &cobra.Command{
		Use:   "list",
		Short: "List tasks in a board (with optional filters)",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := mustFlags(cmd, "board"); err != nil {
				return report(cmd, rf, err)
			}
			ctx := cmd.Context()
			svc, err := rf.openDBWith(ctx, "", board, "")
			if err != nil {
				return report(cmd, rf, err)
			}
			defer svc.Close()

			filter, err := buildTaskFilter(column, priority, dueBefore, dueAfter, search)
			if err != nil {
				return report(cmd, rf, err)
			}
			tasks, err := svc.ListTasks(ctx, filter)
			if err != nil {
				return report(cmd, rf, err)
			}
			if tasks == nil {
				tasks = []*model.Task{}
			}
			return writeOK(cmd, rf, tasks)
		},
	}
	c.Flags().StringVar(&board, "board", "", "Board name (required)")
	c.Flags().StringVar(&column, "column", "", "Filter by column name")
	c.Flags().StringVar(&priority, "priority", "", "Filter by priority (comma-separated)")
	c.Flags().StringVar(&dueBefore, "due-before", "", "Only tasks with due_date <= date (YYYY-MM-DD)")
	c.Flags().StringVar(&dueAfter, "due-after", "", "Only tasks with due_date >= date (YYYY-MM-DD)")
	c.Flags().StringVar(&search, "search", "", "Substring search on title/description")
	return c
}

func buildTaskFilter(column, priorityCSV, dueBefore, dueAfter, search string) (db.TaskFilter, error) {
	filter := db.TaskFilter{ColumnName: column, Search: search}
	if priorityCSV != "" {
		for _, p := range strings.Split(priorityCSV, ",") {
			p = strings.TrimSpace(p)
			if p == "" {
				continue
			}
			pri, err := model.NewPriority(p)
			if err != nil {
				return filter, err
			}
			filter.Priorities = append(filter.Priorities, pri)
		}
	}
	if dueBefore != "" {
		t, err := parseCLIDueDate(dueBefore)
		if err != nil {
			return filter, fmt.Errorf("%w: --due-before: %v", ErrValidation, err)
		}
		filter.DueBefore = &t
	}
	if dueAfter != "" {
		t, err := parseCLIDueDate(dueAfter)
		if err != nil {
			return filter, fmt.Errorf("%w: --due-after: %v", ErrValidation, err)
		}
		filter.DueAfter = &t
	}
	return filter, nil
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
			svc, err := rf.openDB(ctx)
			if err != nil {
				return report(cmd, rf, err)
			}
			defer svc.Close()

			t, err := svc.GetTaskByPrefix(ctx, args[0])
			if err != nil {
				return report(cmd, rf, err)
			}
			return writeOK(cmd, rf, t)
		},
	}
}

func newTaskTreeCmd(rf *rootFlags) *cobra.Command {
	var depth int
	c := &cobra.Command{
		Use:   "tree <id>",
		Short: "Show a task with its descendants (parent/child tree)",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := mustArgs(cmd, args, 1); err != nil {
				return report(cmd, rf, err)
			}
			ctx := cmd.Context()
			svc, err := rf.openDB(ctx)
			if err != nil {
				return report(cmd, rf, err)
			}
			defer svc.Close()

			root, err := svc.GetTaskByPrefix(ctx, args[0])
			if err != nil {
				return report(cmd, rf, err)
			}
			tree, err := buildTaskTree(ctx, svc.DB(), root, depth, 0)
			if err != nil {
				return report(cmd, rf, err)
			}
			return writeOK(cmd, rf, tree)
		},
	}
	c.Flags().IntVar(&depth, "depth", 3, "Maximum nesting depth (0 = unlimited)")
	return c
}

func buildTaskTree(ctx context.Context, conn db.Querier, root *model.Task, maxDepth, current int) (*TaskNode, error) {
	if maxDepth > 0 && current >= maxDepth {
		return &TaskNode{Task: root, Children: nil}, nil
	}
	subs, err := db.ListSubtasks(ctx, conn, root.ID)
	if err != nil {
		return nil, err
	}
	node := &TaskNode{Task: root, Children: make([]*TaskNode, 0, len(subs))}
	for _, sub := range subs {
		child, err := buildTaskTree(ctx, conn, sub, maxDepth, current+1)
		if err != nil {
			return nil, err
		}
		node.Children = append(node.Children, child)
	}
	return node, nil
}

type TaskNode struct {
	Task     *model.Task  `json:"task"`
	Children []*TaskNode  `json:"children,omitempty"`
}

func newTaskMoveCmd(rf *rootFlags) *cobra.Command {
	var board, before, after string
	var top bool
	c := &cobra.Command{
		Use:   "move <id> <column>",
		Short: "Move a task to a column (supports --top/--before/--after)",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := mustArgs(cmd, args, 2); err != nil {
				return report(cmd, rf, err)
			}
			if err := mustFlags(cmd, "board"); err != nil {
				return report(cmd, rf, err)
			}
			ctx := cmd.Context()
			svc, err := rf.openDBWith(ctx, "", board, args[1])
			if err != nil {
				return report(cmd, rf, err)
			}
			defer svc.Close()

			t, err := svc.GetTaskByPrefix(ctx, args[0])
			if err != nil {
				return report(cmd, rf, err)
			}
			r, err := svc.ResolveBoard(ctx)
			if err != nil {
				return report(cmd, rf, err)
			}
			if t.BoardID != r.Board.ID {
				return report(cmd, rf, fmt.Errorf("%w: task belongs to a different board", db.ErrNotFound))
			}
			col, err := db.GetColumnByName(ctx, svc.DB(), r.Board.ID, args[1])
			if err != nil {
				return report(cmd, rf, err)
			}
			beforeID, err := resolveBeforeTarget(ctx, svc, col.ID, top, before, after)
			if err != nil {
				return report(cmd, rf, err)
			}
			if err := svc.MoveTask(ctx, t.ID, col.ID, beforeID); err != nil {
				return report(cmd, rf, err)
			}
			updated, err := svc.GetTask(ctx, t.ID)
			if err != nil {
				return report(cmd, rf, err)
			}
			return writeOK(cmd, rf, updated)
		},
	}
	c.Flags().StringVar(&board, "board", "", "Board name (required)")
	c.Flags().StringVar(&before, "before", "", "Place before this task ID")
	c.Flags().StringVar(&after, "after", "", "Place after this task ID")
	c.Flags().BoolVar(&top, "top", false, "Place at the top of the column")
	return c
}

func resolveBeforeTarget(ctx context.Context, svc *service.Service, colID int64, top bool, before, after string) (*int64, error) {
	switch {
	case top:
		first, err := db.FirstTaskInColumn(ctx, svc.DB(), colID)
		if err != nil {
			return nil, err
		}
		if first == nil {
			return nil, nil
		}
		id := first.ID
		return &id, nil
	case before != "":
		b, err := svc.GetTaskByPrefix(ctx, before)
		if err != nil {
			return nil, err
		}
		id := b.ID
		return &id, nil
	case after != "":
		a, err := svc.GetTaskByPrefix(ctx, after)
		if err != nil {
			return nil, err
		}
		next, err := db.NextTaskInColumn(ctx, svc.DB(), colID, a.ID)
		if err != nil {
			return nil, err
		}
		if next == nil {
			return nil, nil
		}
		id := next.ID
		return &id, nil
	}
	return nil, nil
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
			svc, err := rf.openDB(ctx)
			if err != nil {
				return report(cmd, rf, err)
			}
			defer svc.Close()

			t, err := svc.GetTaskByPrefix(ctx, args[0])
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
				pri, err := model.NewPriority(priority)
				if err != nil {
					return report(cmd, rf, err)
				}
				patch.Priority = &pri
			}
			if clearDue {
				patch.ClearDueDate = true
			} else if due != "" {
				parsed, err := parseCLIDueDate(due)
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

			updated, err := svc.UpdateTask(ctx, t.ID, patch)
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
			svc, err := rf.openDB(ctx)
			if err != nil {
				return report(cmd, rf, err)
			}
			defer svc.Close()

			t, err := svc.GetTaskByPrefix(ctx, args[0])
			if err != nil {
				return report(cmd, rf, err)
			}
			if !force {
				env := Fail(CodeConflict, fmt.Sprintf("refusing to delete task %d without --force", t.ID))
				out := rf.output(cmd.ErrOrStderr())
				_ = out.Write(env)
				return newExitError(ExitConflict, nil)
			}
			if err := svc.DeleteTask(ctx, t.ID); err != nil {
				return report(cmd, rf, err)
			}
			return writeOK(cmd, rf, map[string]any{"deleted": t.ID, "title": t.Title})
		},
	}
	c.Flags().BoolVar(&force, "force", false, "Skip confirmation")
	return c
}

func newTaskHistoryCmd(rf *rootFlags) *cobra.Command {
	var limit int
	c := &cobra.Command{
		Use:   "history <id>",
		Short: "Show the event log for a task",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := mustArgs(cmd, args, 1); err != nil {
				return report(cmd, rf, err)
			}
			ctx := cmd.Context()
			svc, err := rf.openDB(ctx)
			if err != nil {
				return report(cmd, rf, err)
			}
			defer svc.Close()

			t, err := svc.GetTaskByPrefix(ctx, args[0])
			if err != nil {
				return report(cmd, rf, err)
			}
			events, err := db.ListEvents(ctx, svc.DB(), &t.ID, limit)
			if err != nil {
				return report(cmd, rf, err)
			}
			return writeOK(cmd, rf, events)
		},
	}
	c.Flags().IntVar(&limit, "limit", 50, "Maximum events to return")
	return c
}