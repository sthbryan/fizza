package cli

import (
	"fmt"

	"github.com/fizza/fizza/internal/config"
	"github.com/spf13/cobra"
)

func newConfigCmd(rf *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "View and edit user config (mode, default project)",
	}

	cmd.AddCommand(newConfigShowCmd(rf))
	cmd.AddCommand(newConfigSetCmd(rf))
	cmd.AddCommand(newConfigUnsetCmd(rf))
	cmd.AddCommand(newConfigPathCmd(rf))

	return cmd
}

func newConfigShowCmd(rf *rootFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "show",
		Short: "Show effective config (global merged with .fizza)",
		RunE: func(cmd *cobra.Command, args []string) error {
			return writeOK(cmd, rf, rf.conf)
		},
	}
}

func newConfigSetCmd(rf *rootFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Set a config value (mode | project | board)",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := mustArgs(cmd, args, 2); err != nil {
				return report(cmd, rf, err)
			}
			key, value := args[0], args[1]

			cfg, err := config.LoadConfig()
			if err != nil {
				return report(cmd, rf, err)
			}

			switch key {
			case "mode":
				if value != config.ModeLLM && value != config.ModeHuman {
					return report(cmd, rf, fmt.Errorf("%w: mode must be %q or %q, got %q",
						ErrValidation, config.ModeLLM, config.ModeHuman, value))
				}
				cfg.Mode = value
			case "project":
				if value == "" {
					return report(cmd, rf, fmt.Errorf("%w: project value cannot be empty (use `fizza config unset project`)",
						ErrValidation))
				}
				cfg.Project = value
			case "board":
				if value == "" {
					return report(cmd, rf, fmt.Errorf("%w: board value cannot be empty (use `fizza config unset board`)",
						ErrValidation))
				}
				cfg.Board = value
			default:
				return report(cmd, rf, fmt.Errorf("%w: unknown config key %q (want mode, project, or board)",
					ErrValidation, key))
			}

			if err := config.SaveConfig(cfg); err != nil {
				return report(cmd, rf, err)
			}
			return writeOK(cmd, rf, cfg)
		},
	}
}

func newConfigUnsetCmd(rf *rootFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "unset <key>",
		Short: "Unset a config value (project | board)",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := mustArgs(cmd, args, 1); err != nil {
				return report(cmd, rf, err)
			}
			cfg, err := config.LoadConfig()
			if err != nil {
				return report(cmd, rf, err)
			}
			switch args[0] {
			case "project":
				cfg.Project = ""
				cfg.Board = ""
			case "board":
				cfg.Board = ""
			case "mode":
				return report(cmd, rf, fmt.Errorf("%w: mode cannot be unset (use `fizza config set mode llm`)",
					ErrValidation))
			default:
				return report(cmd, rf, fmt.Errorf("%w: unknown config key %q",
					ErrValidation, args[0]))
			}
			if err := config.SaveConfig(cfg); err != nil {
				return report(cmd, rf, err)
			}
			return writeOK(cmd, rf, cfg)
		},
	}
}

func newConfigPathCmd(rf *rootFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "path",
		Short: "Print the config file path",
		RunE: func(cmd *cobra.Command, args []string) error {
			p, err := config.ConfigPath()
			if err != nil {
				return report(cmd, rf, err)
			}
			fmt.Fprintln(cmd.OutOrStdout(), p)
			return nil
		},
	}
}
