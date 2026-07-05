package cli

import (
	"github.com/fizza/fizza/internal/db"
	"github.com/spf13/cobra"
)

func newDoctorCmd(rf *rootFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Run self-checks on the fizza installation",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			svc, err := rf.openDB(ctx)
			if err != nil {
				return report(cmd, rf, err)
			}
			defer svc.Close()
			doctorReport, err := db.Doctor(ctx, svc.DB())
			if err != nil {
				return report(cmd, rf, err)
			}
			return writeOK(cmd, rf, doctorReport)
		},
	}
}