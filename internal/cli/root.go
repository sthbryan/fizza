package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var version = "dev"

func SetVersion(v string) { version = v }

func Execute() error {
	return newRootCmd().Execute()
}

func newRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "fizza",
		Short: "A single-binary kanban for humans and LLMs",
		Long: `Fizza is a SQLite-backed kanban board manager.

All output is structured JSON by default so LLMs and other tools can consume
it directly. Use --pretty for human-readable tables, or --format <json|pretty>
to be explicit.`,
		SilenceUsage:  true,
		SilenceErrors: true,
		Version:       version,
	}

	cmd.PersistentFlags().String("db", "",
		"Path to SQLite database file (overrides FIZZA_DB and default ~/.config/fizza/default.db)")
	cmd.PersistentFlags().String("format", "json",
		"Output format: json | pretty")
	cmd.PersistentFlags().Bool("pretty", false,
		"Shortcut for --format pretty")
	cmd.PersistentFlags().Bool("no-color", false,
		"Disable ANSI colors even in pretty mode")
	cmd.PersistentFlags().Bool("quiet", false,
		"Suppress non-essential output (only the data payload)")

	return cmd
}

func errf(format string, args ...any) error {
	return fmt.Errorf(format, args...)
}
