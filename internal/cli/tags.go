package cli

import (
	"fmt"

	"github.com/fizza/fizza/internal/db"
	"github.com/fizza/fizza/internal/model"
	"github.com/spf13/cobra"
)

func newTagCmd(rf *rootFlags) *cobra.Command {
	cmd := &cobra.Command{Use: "tag", Short: "Manage tags (labels) for tasks"}
	cmd.AddCommand(newTagAddCmd(rf))
	cmd.AddCommand(newTagListCmd(rf))
	cmd.AddCommand(newTagDeleteCmd(rf))
	cmd.AddCommand(newTagAttachCmd(rf))
	cmd.AddCommand(newTagDetachCmd(rf))
	return cmd
}

func newTagAddCmd(rf *rootFlags) *cobra.Command {
	var project string
	c := &cobra.Command{
		Use:   "add <name>",
		Short: "Create a tag in the current project",
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
			if project != "" {
				p, err = db.GetProjectByName(ctx, svc.DB(), project)
				if err != nil {
					return report(cmd, rf, err)
				}
			}
			t, err := db.CreateTag(ctx, svc.DB(), p.ID, args[0])
			if err != nil {
				return report(cmd, rf, err)
			}
			return writeOK(cmd, rf, t)
		},
	}
	c.Flags().StringVar(&project, "project", "", "Override project (defaults to current)")
	return c
}

func newTagListCmd(rf *rootFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List tags in the current project",
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
			tags, err := db.ListTags(ctx, svc.DB(), p.ID)
			if err != nil {
				return report(cmd, rf, err)
			}
			if tags == nil {
				tags = []*model.Tag{}
			}
			return writeOK(cmd, rf, tags)
		},
	}
}

func newTagDeleteCmd(rf *rootFlags) *cobra.Command {
	var force bool
	c := &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a tag (cascades from tasks)",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := mustArgs(cmd, args, 1); err != nil {
				return report(cmd, rf, err)
			}
			id, err := parseInt64(args[0])
			if err != nil {
				return report(cmd, rf, fmt.Errorf("%w: %v", ErrValidation, err))
			}
			if !force {
				env := Fail(CodeConflict, fmt.Sprintf("refusing to delete tag %d without --force", id))
				out := rf.output(cmd.ErrOrStderr())
				_ = out.Write(env)
				return newExitError(ExitConflict, nil)
			}
			ctx := cmd.Context()
			svc, err := rf.openDB(ctx)
			if err != nil {
				return report(cmd, rf, err)
			}
			defer svc.Close()
			if err := db.DeleteTag(ctx, svc.DB(), id); err != nil {
				return report(cmd, rf, err)
			}
			return writeOK(cmd, rf, map[string]any{"deleted": id})
		},
	}
	c.Flags().BoolVar(&force, "force", false, "Skip confirmation")
	return c
}

func newTagAttachCmd(rf *rootFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "attach <task-id> <tag-id>",
		Short: "Attach a tag to a task",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := mustArgs(cmd, args, 2); err != nil {
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
			tagID, err := parseInt64(args[1])
			if err != nil {
				return report(cmd, rf, fmt.Errorf("%w: %v", ErrValidation, err))
			}
			if err := db.AddTagToTask(ctx, svc.DB(), t.ID, tagID); err != nil {
				return report(cmd, rf, err)
			}
			return writeOK(cmd, rf, map[string]any{"task_id": t.ID, "tag_id": tagID})
		},
	}
}

func newTagDetachCmd(rf *rootFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "detach <task-id> <tag-id>",
		Short: "Detach a tag from a task",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := mustArgs(cmd, args, 2); err != nil {
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
			tagID, err := parseInt64(args[1])
			if err != nil {
				return report(cmd, rf, fmt.Errorf("%w: %v", ErrValidation, err))
			}
			if err := db.RemoveTagFromTask(ctx, svc.DB(), t.ID, tagID); err != nil {
				return report(cmd, rf, err)
			}
			return writeOK(cmd, rf, map[string]any{"task_id": t.ID, "tag_id": tagID})
		},
	}
}
