package cli

import (
	"fmt"
	"os"
	"strings"

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
	var project, columns string
	c := &cobra.Command{
		Use:   "create <name>",
		Short: "Create a board in a project",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			conn, err := rf.openDB(ctx)
			if err != nil {
				return report(cmd, rf, err)
			}
			defer conn.Close()

			p, err := db.GetProjectByName(ctx, conn, project)
			if err != nil {
				return report(cmd, rf, err)
			}

			var cols []string
			if columns != "" {
				for _, c := range strings.Split(columns, ",") {
					c = strings.TrimSpace(c)
					if c != "" {
						cols = append(cols, c)
					}
				}
			}

			b, err := db.CreateBoardWithColumns(ctx, conn, p.ID, args[0], cols)
			if err != nil {
				return report(cmd, rf, err)
			}
			return writeOK(cmd, rf, b)
		},
	}
	c.Flags().StringVar(&project, "project", "", "Project name (required)")
	c.Flags().StringVar(&columns, "columns", "", "Comma-separated column names (default: todo,in_progress,done)")
	_ = c.MarkFlagRequired("project")
	return c
}

func newBoardListCmd(rf *rootFlags) *cobra.Command {
	var project string
	c := &cobra.Command{
		Use:   "list",
		Short: "List boards in a project",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			conn, err := rf.openDB(ctx)
			if err != nil {
				return report(cmd, rf, err)
			}
			defer conn.Close()

			p, err := db.GetProjectByName(ctx, conn, project)
			if err != nil {
				return report(cmd, rf, err)
			}
			boards, err := db.ListBoards(ctx, conn, p.ID)
			if err != nil {
				return report(cmd, rf, err)
			}
			if boards == nil {
				boards = []*model.Board{}
			}
			return writeOK(cmd, rf, boards)
		},
	}
	c.Flags().StringVar(&project, "project", "", "Project name (required)")
	_ = c.MarkFlagRequired("project")
	return c
}

func newBoardShowCmd(rf *rootFlags) *cobra.Command {
	var project string
	c := &cobra.Command{
		Use:   "show <name>",
		Short: "Show board with columns",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			conn, err := rf.openDB(ctx)
			if err != nil {
				return report(cmd, rf, err)
			}
			defer conn.Close()

			p, err := db.GetProjectByName(ctx, conn, project)
			if err != nil {
				return report(cmd, rf, err)
			}
			boards, err := db.ListBoards(ctx, conn, p.ID)
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
				return report(cmd, rf, fmt.Errorf("%w: board %q in project %q", db.ErrNotFound, args[0], project))
			}
			cols, err := db.ListColumns(ctx, conn, found.ID)
			if err != nil {
				return report(cmd, rf, err)
			}
			if cols == nil {
				cols = []*model.Column{}
			}
			return writeOK(cmd, rf, map[string]any{"board": found, "columns": cols})
		},
	}
	c.Flags().StringVar(&project, "project", "", "Project name (required)")
	_ = c.MarkFlagRequired("project")
	return c
}

func newBoardDeleteCmd(rf *rootFlags) *cobra.Command {
	var project string
	var force bool
	c := &cobra.Command{
		Use:   "delete <name>",
		Short: "Delete a board (blocked if tasks exist)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			conn, err := rf.openDB(ctx)
			if err != nil {
				return report(cmd, rf, err)
			}
			defer conn.Close()

			p, err := db.GetProjectByName(ctx, conn, project)
			if err != nil {
				return report(cmd, rf, err)
			}
			boards, err := db.ListBoards(ctx, conn, p.ID)
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
				return report(cmd, rf, fmt.Errorf("%w: board %q in project %q", db.ErrNotFound, args[0], project))
			}

			if !force {
				env := Fail(CodeConflict, fmt.Sprintf("refusing to delete %q without --force", args[0]))
				out := rf.output(cmd.ErrOrStderr())
				_ = out.Write(env)
				os.Exit(ExitConflict)
			}

			if err := db.DeleteBoard(ctx, conn, found.ID); err != nil {
				return report(cmd, rf, err)
			}
			return writeOK(cmd, rf, map[string]any{"deleted": args[0], "id": found.ID})
		},
	}
	c.Flags().StringVar(&project, "project", "", "Project name (required)")
	c.Flags().BoolVar(&force, "force", false, "Skip confirmation")
	_ = c.MarkFlagRequired("project")
	return c
}