package cli

import (
	"github.com/fizza/fizza/internal/db"
	"github.com/spf13/cobra"
)

func newSchemaCmd(rf *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "schema",
		Short: "Inspect DB schema, applied migrations, and integrity",
	}
	cmd.AddCommand(newSchemaShowCmd(rf))
	cmd.AddCommand(newSchemaMigrationsCmd(rf))
	return cmd
}

func newSchemaShowCmd(rf *rootFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "show",
		Short: "Show all tables, indexes, and their DDL",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			svc, err := rf.openDB(ctx)
			if err != nil {
				return report(cmd, rf, err)
			}
			defer svc.Close()
			ddl, err := db.Schema(ctx, svc.DB())
			if err != nil {
				return report(cmd, rf, err)
			}
			return writeOK(cmd, rf, ddl)
		},
	}
}

func newSchemaMigrationsCmd(rf *rootFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "migrations",
		Short: "Show applied and pending migrations with checksums",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			svc, err := rf.openDB(ctx)
			if err != nil {
				return report(cmd, rf, err)
			}
			defer svc.Close()
			pending, applied, err := db.MigrationStatus(ctx, svc.DB())
			if err != nil {
				return report(cmd, rf, err)
			}
			return writeOK(cmd, rf, map[string]any{
				"applied": applied,
				"pending": pending,
			})
		},
	}
}