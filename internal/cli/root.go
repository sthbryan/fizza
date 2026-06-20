package cli

import (
	"context"
	"database/sql"
	"io"

	"github.com/fizza/fizza/internal/config"
	"github.com/fizza/fizza/internal/db"
	"github.com/spf13/cobra"
)

var version = "dev"

func SetVersion(v string) { version = v }

func Execute() error {
	return newRootCmd().ExecuteContext(context.Background())
}

type rootFlags struct {
	dbPath  string
	format  string
	pretty  bool
	noColor bool
}

func (rf *rootFlags) output(w io.Writer) *Output {
	format := rf.format
	if rf.pretty {
		format = "pretty"
	}
	noColor := rf.noColor || !StdoutIsTTY()
	return NewOutput(w, format, noColor)
}

func (rf *rootFlags) openDB(ctx context.Context) (*sql.DB, error) {
	path, err := config.DBPath(rf.dbPath)
	if err != nil {
		return nil, err
	}
	return db.Open(ctx, path)
}

func newRootCmd() *cobra.Command {
	rf := &rootFlags{}
	cmd := &cobra.Command{
		Use:           "fizza",
		Short:         "Single-binary kanban for humans and LLMs",
		SilenceUsage:  true,
		SilenceErrors: true,
		Version:       version,
	}

	cmd.PersistentFlags().StringVar(&rf.dbPath, "db", "", "SQLite path (overrides FIZZA_DB)")
	cmd.PersistentFlags().StringVar(&rf.format, "format", "json", "Output format: json|pretty")
	cmd.PersistentFlags().BoolVar(&rf.pretty, "pretty", false, "Shortcut for --format pretty")
	cmd.PersistentFlags().BoolVar(&rf.noColor, "no-color", false, "Disable ANSI colors")

	cmd.AddCommand(newProjectCmd(rf))
	cmd.AddCommand(newBoardCmd(rf))
	cmd.AddCommand(newTaskCmd(rf))

	return cmd
}