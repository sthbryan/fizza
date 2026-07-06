package cli

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var ErrValidation = errors.New("validation")

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
