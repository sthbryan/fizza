package cli

import (
	"fmt"

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
	return cmd
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