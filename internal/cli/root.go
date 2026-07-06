package cli

import (
	"context"
	"errors"
	"io"
	"os"

	"github.com/fizza/fizza/internal/config"
	"github.com/fizza/fizza/internal/db"
	"github.com/fizza/fizza/internal/service"
	"github.com/spf13/cobra"
)

var version = "dev"

func SetVersion(v string) { version = v }

func Execute() error {
	return newRootCmd().ExecuteContext(context.Background())
}

func ExecuteWithCode() (int, error) {
	err := Execute()
	if err == nil {
		return ExitOK, nil
	}
	var ee *ExitError
	if errors.As(err, &ee) {
		return ee.Code, ee.Err
	}
	return ExitGeneric, err
}

type rootFlags struct {
	format  string
	noColor bool
	conf    config.Config
}

func (rf *rootFlags) output(w io.Writer) *Output {
	format := config.ResolveMode(rf.format, rf.conf.Mode)
	noColor := rf.noColor || !StdoutIsTTY()
	return NewOutput(w, format, noColor)
}

func (rf *rootFlags) openDB(ctx context.Context) (*service.Service, error) {
	path, err := config.DBPath()
	if err != nil {
		return nil, err
	}
	conn, err := db.Open(ctx, path)
	if err != nil {
		return nil, err
	}
	project := rf.conf.Project
	if cmd := currentCmdProject(); cmd != "" {
		project = cmd
	}
	return service.New(conn, project, "", ""), nil
}

func (rf *rootFlags) openDBWith(ctx context.Context, project, board, column string) (*service.Service, error) {
	path, err := config.DBPath()
	if err != nil {
		return nil, err
	}
	conn, err := db.Open(ctx, path)
	if err != nil {
		return nil, err
	}
	if project == "" {
		project = rf.conf.Project
	}
	if board == "" {
		board = rf.conf.Board
	}
	return service.New(conn, project, board, column), nil
}

func currentCmdProject() string {
	return os.Getenv("FIZZA_CMD_PROJECT")
}

func newRootCmd() *cobra.Command {
	rf := &rootFlags{}
	cmd := &cobra.Command{
		Use:           "fizza",
		Short:         "Single-binary kanban for humans and LLMs",
		SilenceUsage:  true,
		SilenceErrors: true,
		Version:       version,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			name := cmd.Name()
			if name == "fizza" || name == "mcp" || name == "help" {
				return nil
			}
			cwd, _ := os.Getwd()
			cfg, err := config.LoadEffectiveConfig(cwd)
			if err != nil {
				return err
			}
			rf.conf = cfg
			return nil
		},
	}

	cmd.PersistentFlags().StringVar(&rf.format, "format", "json", "Output format: json (default) or pretty (human tables)")
	cmd.PersistentFlags().BoolVar(&rf.noColor, "no-color", false, "Disable ANSI colors")

	cmd.AddCommand(newProjectCmd(rf))
	cmd.AddCommand(newBoardCmd(rf))
	cmd.AddCommand(newTaskCmd(rf))
	cmd.AddCommand(newTagCmd(rf))
	cmd.AddCommand(newConfigCmd(rf))
	cmd.AddCommand(newSchemaCmd(rf))
	cmd.AddCommand(newDoctorCmd(rf))
	cmd.AddCommand(newMCPCmd(rf))
	cmd.AddCommand(newServeCmd(rf))

	return cmd
}