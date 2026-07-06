package cli

import (
	"context"
	"fmt"

	"github.com/fizza/fizza/internal/config"
	"github.com/fizza/fizza/internal/db"
	"github.com/fizza/fizza/internal/model"
	"github.com/spf13/cobra"
)

func newBoardCmd(rf *rootFlags) *cobra.Command {
	cmd := &cobra.Command{Use: "board", Short: "Manage boards"}
	cmd.AddCommand(newBoardCreateCmd(rf))
	cmd.AddCommand(newBoardListCmd(rf))
	cmd.AddCommand(newBoardShowCmd(rf))
	cmd.AddCommand(newBoardDeleteCmd(rf))
	cmd.AddCommand(newBoardSetWIPCmd(rf))
	cmd.AddCommand(newBoardSetCmd(rf))
	return cmd
}

func newBoardSetCmd(rf *rootFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "set <name>",
		Short: "Set the default board for this project (shortcut for `fizza config set board <name>`)",
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
			p, err := svc.ResolveProject(ctx)
			if err != nil {
				return report(cmd, rf, err)
			}
			if _, err := findBoardInBoard(ctx, svc.DB(), p.ID, args[0]); err != nil {
				return report(cmd, rf, err)
			}
			cfg, err := config.LoadConfig()
			if err != nil {
				return report(cmd, rf, err)
			}
			cfg.Project = p.Name
			cfg.Board = args[0]
			if err := config.SaveConfig(cfg); err != nil {
				return report(cmd, rf, err)
			}
			return writeOK(cmd, rf, cfg)
		},
	}
}

func newBoardSetWIPCmd(rf *rootFlags) *cobra.Command {
	var limit int
	var clear bool
	c := &cobra.Command{
		Use:   "set-wip <board> <column>",
		Short: "Set or clear the WIP limit on a column",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := mustArgs(cmd, args, 2); err != nil {
				return report(cmd, rf, err)
			}
			if clear && cmd.Flags().Changed("limit") {
				return report(cmd, rf, fmt.Errorf("%w: --clear and --limit are mutually exclusive", ErrValidation))
			}
			ctx := cmd.Context()
			svc, err := rf.openDB(ctx)
			if err != nil {
				return report(cmd, rf, err)
			}
			defer svc.Close()
			p, err := svc.ResolveProject(ctx)
			if err != nil {
				return report(cmd, rf, err)
			}
			col, err := findColumnInBoard(ctx, svc.DB(), p.ID, args[0], args[1])
			if err != nil {
				return report(cmd, rf, err)
			}
			var lim *int
			if !clear {
				if limit < 0 {
					return report(cmd, rf, fmt.Errorf("%w: --limit must be >= 0", ErrValidation))
				}
				lim = &limit
			}
			if err := db.UpdateColumnWIPLimit(ctx, svc.DB(), col.ID, lim); err != nil {
				return report(cmd, rf, err)
			}
			return writeOK(cmd, rf, map[string]any{
				"board":     args[0],
				"column":    args[1],
				"wip_limit": lim,
			})
		},
	}
	c.Flags().IntVar(&limit, "limit", 0, "WIP limit (>= 0)")
	c.Flags().BoolVar(&clear, "clear", false, "Remove the WIP limit")
	return c
}

func findColumnInBoard(ctx context.Context, conn db.Querier, projectID int64, board, column string) (*model.Column, error) {
	b, err := findBoardInBoard(ctx, conn, projectID, board)
	if err != nil {
		return nil, err
	}
	return db.GetColumnByName(ctx, conn, b.ID, column)
}

func findBoardInBoard(ctx context.Context, conn db.Querier, projectID int64, board string) (*model.Board, error) {
	boards, err := db.ListBoards(ctx, conn, projectID)
	if err != nil {
		return nil, err
	}
	for _, b := range boards {
		if b.Name == board {
			return b, nil
		}
	}
	return nil, fmt.Errorf("%w: board %q in project", db.ErrNotFound, board)
}

func newBoardCreateCmd(rf *rootFlags) *cobra.Command {
	var columns string
	c := &cobra.Command{
		Use:   "create <name>",
		Short: "Create a board in a project",
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

			p, err := svc.ResolveProject(ctx)
			if err != nil {
				return report(cmd, rf, err)
			}

			b, err := db.CreateBoardWithColumns(ctx, svc.DB(), p.ID, args[0], splitColumnsCLI(columns))
			if err != nil {
				return report(cmd, rf, err)
			}
			return writeOK(cmd, rf, b)
		},
	}
	c.Flags().StringVar(&columns, "columns", "", "Comma-separated column names (default: todo,in_progress,done)")
	return c
}

func newBoardListCmd(rf *rootFlags) *cobra.Command {
	c := &cobra.Command{
		Use:   "list",
		Short: "List boards in a project",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			svc, err := rf.openDB(ctx)
			if err != nil {
				return report(cmd, rf, err)
			}
			defer svc.Close()

			p, err := svc.ResolveProject(ctx)
			if err != nil {
				return report(cmd, rf, err)
			}
			boards, err := db.ListBoards(ctx, svc.DB(), p.ID)
			if err != nil {
				return report(cmd, rf, err)
			}
			if boards == nil {
				boards = []*model.Board{}
			}
			return writeOK(cmd, rf, boards)
		},
	}
	return c
}

func newBoardShowCmd(rf *rootFlags) *cobra.Command {
	c := &cobra.Command{
		Use:   "show <name>",
		Short: "Show board with columns and tasks",
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

			p, err := svc.ResolveProject(ctx)
			if err != nil {
				return report(cmd, rf, err)
			}
			boards, err := db.ListBoards(ctx, svc.DB(), p.ID)
			if err != nil {
				return report(cmd, rf, err)
			}
			var found *model.Board
			for _, b := range boards {
				if b.Name == args[0] {
					found = b
					break
				}
			}
			if found == nil {
				return report(cmd, rf, fmt.Errorf("%w: board %q in project %q", db.ErrNotFound, args[0], p.Name))
			}
			cols, err := db.ListColumns(ctx, svc.DB(), found.ID)
			if err != nil {
				return report(cmd, rf, err)
			}
			tasks, err := db.ListTasksInBoard(ctx, svc.DB(), found.ID, db.TaskFilter{})
			if err != nil {
				return report(cmd, rf, err)
			}
			buckets := make([]BoardColumnBucket, len(cols))
			byCol := map[int64]int{}
			for i, c := range cols {
				byCol[c.ID] = i
				buckets[i] = BoardColumnBucket{Column: c, Tasks: []*model.Task{}}
			}
			for _, t := range tasks {
				if idx, ok := byCol[t.ColumnID]; ok {
					buckets[idx].Tasks = append(buckets[idx].Tasks, t)
				}
			}
			return writeOK(cmd, rf, BoardView{Board: found, Columns: buckets})
		},
	}
	return c
}

func newBoardDeleteCmd(rf *rootFlags) *cobra.Command {
	var force bool
	c := &cobra.Command{
		Use:   "delete <name>",
		Short: "Delete a board (blocked if tasks exist)",
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

			p, err := svc.ResolveProject(ctx)
			if err != nil {
				return report(cmd, rf, err)
			}
			boards, err := db.ListBoards(ctx, svc.DB(), p.ID)
			if err != nil {
				return report(cmd, rf, err)
			}
			var found *model.Board
			for _, b := range boards {
				if b.Name == args[0] {
					found = b
					break
				}
			}
			if found == nil {
				return report(cmd, rf, fmt.Errorf("%w: board %q in project %q", db.ErrNotFound, args[0], p.Name))
			}

			if !force {
				env := Fail(CodeConflict, fmt.Sprintf("refusing to delete %q without --force", args[0]))
				out := rf.output(cmd.ErrOrStderr())
				_ = out.Write(env)
				return newExitError(ExitConflict, nil)
			}

			if err := db.DeleteBoard(ctx, svc.DB(), found.ID); err != nil {
				return report(cmd, rf, err)
			}
			return writeOK(cmd, rf, map[string]any{"deleted": args[0], "id": found.ID})
		},
	}
	c.Flags().BoolVar(&force, "force", false, "Skip confirmation")
	return c
}

func splitColumnsCLI(s string) []string {
	if s == "" {
		return nil
	}
	return splitCSVSimple(s)
}

func splitCSVSimple(s string) []string {
	out := []string{}
	current := ""
	for _, r := range s {
		if r == ',' {
			if t := trimSpaces(current); t != "" {
				out = append(out, t)
			}
			current = ""
			continue
		}
		current += string(r)
	}
	if t := trimSpaces(current); t != "" {
		out = append(out, t)
	}
	return out
}

func trimSpaces(s string) string {
	start := 0
	end := len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t') {
		end--
	}
	return s[start:end]
}
