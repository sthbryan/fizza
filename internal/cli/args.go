package cli

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var ErrValidation = errors.New("validation")

type runConfig struct {
	args     int
	argsMin  int
	argsMax  int
	required []string
}

func (rc *runConfig) check(cmd *cobra.Command, args []string) error {
	switch {
	case rc.args > 0 && len(args) != rc.args:
		return mustArgs(cmd, args, rc.args)
	case rc.argsMin > 0 || rc.argsMax > 0:
		if rc.argsMax == 0 {
			rc.argsMax = rc.argsMin
		}
		if err := mustRange(cmd, args, rc.argsMin, rc.argsMax); err != nil {
			return err
		}
	}
	if len(rc.required) > 0 {
		return mustFlags(cmd, rc.required...)
	}
	return nil
}

func mustArgs(cmd *cobra.Command, args []string, n int) error {
	if len(args) == n {
		return nil
	}
	use := strings.TrimSpace(cmd.UseLine())
	switch {
	case len(args) == 0 && n == 1:
		return fmt.Errorf("%w: %s requires a %s argument; usage: %s",
			ErrValidation, cmd.CommandPath(), cmd.Name(), use)
	case len(args) < n:
		return fmt.Errorf("%w: %s requires %d argument(s), got %d; usage: %s",
			ErrValidation, cmd.CommandPath(), n, len(args), use)
	default:
		return fmt.Errorf("%w: %s accepts %d argument(s), got %d; usage: %s",
			ErrValidation, cmd.CommandPath(), n, len(args), use)
	}
}

func mustRange(cmd *cobra.Command, args []string, min, max int) error {
	if len(args) >= min && len(args) <= max {
		return nil
	}
	use := strings.TrimSpace(cmd.UseLine())
	return fmt.Errorf("%w: %s expects between %d and %d argument(s), got %d; usage: %s",
		ErrValidation, cmd.CommandPath(), min, max, len(args), use)
}

func mustFlags(cmd *cobra.Command, flagNames ...string) error {
	var missing []string
	for _, name := range flagNames {
		if !cmd.Flags().Changed(name) {
			missing = append(missing, "--"+name)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("%w: %s requires %s; usage: %s",
			ErrValidation, cmd.CommandPath(), strings.Join(missing, ", "), cmd.UseLine())
	}
	return nil
}
