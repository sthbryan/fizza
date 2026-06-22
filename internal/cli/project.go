package cli

import (
	"fmt"
	"os"

	"github.com/fizza/fizza/internal/config"
	"github.com/fizza/fizza/internal/db"
	"github.com/fizza/fizza/internal/model"
	"github.com/spf13/cobra"
)

func newProjectCmd(rf *rootFlags) *cobra.Command {
	cmd := &cobra.Command{Use: "project", Short: "Manage projects"}
	cmd.AddCommand(newProjectNewCmd(rf))
	cmd.AddCommand(newProjectListCmd(rf))
	cmd.AddCommand(newProjectShowCmd(rf))
	cmd.AddCommand(newProjectDeleteCmd(rf))
	cmd.AddCommand(newProjectSetCmd(rf))
	return cmd
}

func newProjectSetCmd(rf *rootFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "set <name>",
		Short: "Set the default project (used when --project is omitted)",
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

			if _, err := db.GetProjectByName(ctx, conn, args[0]); err != nil {
				return report(cmd, rf, err)
			}

			cfg, err := config.LoadConfig()
			if err != nil {
				return report(cmd, rf, err)
			}
			cfg.Project = args[0]
			if err := config.SaveConfig(cfg); err != nil {
				return report(cmd, rf, err)
			}
			return writeOK(cmd, rf, map[string]any{"default_project": args[0]})
		},
	}
}

func newProjectNewCmd(rf *rootFlags) *cobra.Command {
	var desc string
	c := &cobra.Command{
		Use:   "new <name>",
		Short: "Create a new project",
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

			p, err := db.CreateProject(ctx, conn, args[0], desc)
			if err != nil {
				return report(cmd, rf, err)
			}
			return writeOK(cmd, rf, p)
		},
	}
	c.Flags().StringVar(&desc, "desc", "", "Project description")
	return c
}

func newProjectListCmd(rf *rootFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all projects",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			conn, err := rf.openDB(ctx)
			if err != nil {
				return report(cmd, rf, err)
			}
			defer conn.Close()

			projects, err := db.ListProjects(ctx, conn)
			if err != nil {
				return report(cmd, rf, err)
			}
			if projects == nil {
				projects = []*model.Project{}
			}
			return writeOK(cmd, rf, projects)
		},
	}
}

func newProjectShowCmd(rf *rootFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "show <name>",
		Short: "Show project details",
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

			p, err := db.GetProjectByName(ctx, conn, args[0])
			if err != nil {
				return report(cmd, rf, err)
			}
			return writeOK(cmd, rf, p)
		},
	}
}

func newProjectDeleteCmd(rf *rootFlags) *cobra.Command {
	var force bool
	c := &cobra.Command{
		Use:   "delete <name>",
		Short: "Delete a project (cascades boards, columns, tasks)",
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

			p, err := db.GetProjectByName(ctx, conn, args[0])
			if err != nil {
				return report(cmd, rf, err)
			}
			if !force {
				env := Fail(CodeConflict, fmt.Sprintf("refusing to delete %q without --force", args[0]))
				out := rf.output(cmd.ErrOrStderr())
				if werr := out.Write(env); werr != nil {
					return werr
				}
				os.Exit(ExitConflict)
			}
			if err := db.DeleteProject(ctx, conn, p.ID); err != nil {
				return report(cmd, rf, err)
			}
			return writeOK(cmd, rf, map[string]any{"deleted": args[0], "id": p.ID})
		},
	}
	c.Flags().BoolVar(&force, "force", false, "Skip confirmation")
	return c
}

func writeOK(cmd *cobra.Command, rf *rootFlags, data any) error {
	out := rf.output(cmd.OutOrStdout())
	return out.Write(OK(data))
}

func report(cmd *cobra.Command, rf *rootFlags, err error) error {
	env, exit := ClassifyError(err)
	out := rf.output(cmd.ErrOrStderr())
	_ = out.Write(env)
	os.Exit(exit)
	return nil
}