package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/fizza/fizza/internal/config"
	"github.com/fizza/fizza/internal/db"
	"github.com/fizza/fizza/internal/model"
	"github.com/fizza/fizza/internal/service"
	"github.com/spf13/cobra"
)

func newProjectCmd(rf *rootFlags) *cobra.Command {
	cmd := &cobra.Command{Use: "project", Short: "Manage projects"}
	cmd.AddCommand(newProjectNewCmd(rf))
	cmd.AddCommand(newProjectListCmd(rf))
	cmd.AddCommand(newProjectShowCmd(rf))
	cmd.AddCommand(newProjectDeleteCmd(rf))
	cmd.AddCommand(newProjectSetCmd(rf))
	cmd.AddCommand(newProjectExportCmd(rf))
	cmd.AddCommand(newProjectImportCmd(rf))
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
			svc, err := rf.openDB(ctx)
			if err != nil {
				return report(cmd, rf, err)
			}
			defer svc.DB().Close()

			if _, err := db.GetProjectByName(ctx, svc.DB(), args[0]); err != nil {
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
			svc, err := rf.openDB(ctx)
			if err != nil {
				return report(cmd, rf, err)
			}
			defer svc.DB().Close()

			p, err := db.CreateProject(ctx, svc.DB(), args[0], desc)
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
			svc, err := rf.openDB(ctx)
			if err != nil {
				return report(cmd, rf, err)
			}
			defer svc.DB().Close()

			projects, err := db.ListProjects(ctx, svc.DB())
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
			svc, err := rf.openDB(ctx)
			if err != nil {
				return report(cmd, rf, err)
			}
			defer svc.DB().Close()

			p, err := db.GetProjectByName(ctx, svc.DB(), args[0])
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
			svc, err := rf.openDB(ctx)
			if err != nil {
				return report(cmd, rf, err)
			}
			defer svc.DB().Close()

			p, err := db.GetProjectByName(ctx, svc.DB(), args[0])
			if err != nil {
				return report(cmd, rf, err)
			}
			if !force {
				env := Fail(CodeConflict, fmt.Sprintf("refusing to delete %q without --force", args[0]))
				out := rf.output(cmd.ErrOrStderr())
				if werr := out.Write(env); werr != nil {
					return werr
				}
				return newExitError(ExitConflict, nil)
			}
			if err := db.DeleteProject(ctx, svc.DB(), p.ID); err != nil {
				return report(cmd, rf, err)
			}
			return writeOK(cmd, rf, map[string]any{"deleted": args[0], "id": p.ID})
		},
	}
	c.Flags().BoolVar(&force, "force", false, "Skip confirmation")
	return c
}

func newProjectExportCmd(rf *rootFlags) *cobra.Command {
	c := &cobra.Command{
		Use:   "export <name>",
		Short: "Export a project (boards, columns, tasks) as JSON to stdout",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := mustArgs(cmd, args, 1); err != nil {
				return report(cmd, rf, err)
			}
			ctx := cmd.Context()
			svc, err := rf.openDB(ctx)
			if err != nil {
				return report(cmd, rf, err)
			}
			defer svc.DB().Close()

			p, err := db.GetProjectByName(ctx, svc.DB(), args[0])
			if err != nil {
				return report(cmd, rf, err)
			}
			boards, err := db.ListBoards(ctx, svc.DB(), p.ID)
			if err != nil {
				return report(cmd, rf, err)
			}
			out := ExportPayload{
				Project: p,
				ExportedAt: time.Now().UTC(),
			}
			for _, b := range boards {
				cols, err := db.ListColumns(ctx, svc.DB(), b.ID)
				if err != nil {
					return report(cmd, rf, err)
				}
				tasks, err := db.ListTasksInBoard(ctx, svc.DB(), b.ID, db.TaskFilter{})
				if err != nil {
					return report(cmd, rf, err)
				}
				out.Boards = append(out.Boards, ExportedBoard{Board: b, Columns: cols, Tasks: tasks})
			}
			if cmd.Flag("format").Value.String() != "" && cmd.Flag("format").Value.String() != "json" {
				return report(cmd, rf, fmt.Errorf("%w: export only supports JSON format", ErrValidation))
			}
			b, err := json.MarshalIndent(out, "", "  ")
			if err != nil {
				return report(cmd, rf, err)
			}
			if _, err := fmt.Fprintln(cmd.OutOrStdout(), string(b)); err != nil {
				return report(cmd, rf, err)
			}
			return nil
		},
	}
	return c
}

func newProjectImportCmd(rf *rootFlags) *cobra.Command {
	var fromFile string
	c := &cobra.Command{
		Use:   "import",
		Short: "Import a project from JSON (produced by `fizza project export`)",
		RunE: func(cmd *cobra.Command, args []string) error {
			if fromFile == "" {
				return report(cmd, rf, fmt.Errorf("%w: --from is required", ErrValidation))
			}
			data, err := os.ReadFile(fromFile)
			if err != nil {
				return report(cmd, rf, err)
			}
			var payload ExportPayload
			if err := json.Unmarshal(data, &payload); err != nil {
				return report(cmd, rf, fmt.Errorf("invalid export file: %w", err))
			}
			if payload.Project == nil {
				return report(cmd, rf, fmt.Errorf("%w: missing project in export", ErrValidation))
			}
			ctx := cmd.Context()
			svc, err := rf.openDB(ctx)
			if err != nil {
				return report(cmd, rf, err)
			}
			defer svc.DB().Close()

			existing, err := db.GetProjectByName(ctx, svc.DB(), payload.Project.Name)
			if err == nil && existing != nil {
				return report(cmd, rf, fmt.Errorf("%w: project %q already exists", db.ErrDuplicate, payload.Project.Name))
			} else if err != nil && !db.IsNotFound(err) {
				return report(cmd, rf, err)
			}

			created, err := db.CreateProject(ctx, svc.DB(), payload.Project.Name, payload.Project.Description)
			if err != nil {
				return report(cmd, rf, err)
			}
			imported := map[int64]int64{}
			for _, eb := range payload.Boards {
				b, err := db.CreateBoardWithColumns(ctx, svc.DB(), created.ID, eb.Board.Name, nil)
				if err != nil {
					return report(cmd, rf, err)
				}
				colByOld := map[int64]int64{}
				cols, err := db.ListColumns(ctx, svc.DB(), b.ID)
				if err != nil {
					return report(cmd, rf, err)
				}
				for _, old := range eb.Columns {
					for _, newC := range cols {
						if newC.Name == old.Name {
							colByOld[old.ID] = newC.ID
							break
						}
					}
				}
				imported[eb.Board.ID] = b.ID
				for _, t := range eb.Tasks {
					t.BoardID = b.ID
					if newCol, ok := colByOld[t.ColumnID]; ok {
						t.ColumnID = newCol
					}
					if err := db.CreateTask(ctx, svc.DB(), t); err != nil {
						return report(cmd, rf, err)
					}
				}
			}
			_ = strings.TrimSpace
			return writeOK(cmd, rf, map[string]any{
				"imported_project": created.Name,
				"boards_imported":  len(payload.Boards),
			})
		},
	}
	c.Flags().StringVar(&fromFile, "from", "", "Path to JSON export file (required)")
	return c
}

type ExportPayload struct {
	Project    *model.Project   `json:"project"`
	ExportedAt time.Time        `json:"exported_at"`
	Boards     []ExportedBoard  `json:"boards"`
}

type ExportedBoard struct {
	Board   *model.Board    `json:"board"`
	Columns []*model.Column `json:"columns"`
	Tasks   []*model.Task   `json:"tasks"`
}

func writeOK(cmd *cobra.Command, rf *rootFlags, data any) error {
	out := rf.output(cmd.OutOrStdout())
	return out.Write(OK(data))
}

func report(cmd *cobra.Command, rf *rootFlags, err error) error {
	env, exit := ClassifyError(err)
	out := rf.output(cmd.ErrOrStderr())
	_ = out.Write(env)
	return newExitError(exit, err)
}

var _ = service.ErrValidation