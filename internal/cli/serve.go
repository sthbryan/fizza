package cli

import (
	"fmt"
	"time"

	"github.com/fizza/fizza/internal/config"
	"github.com/fizza/fizza/internal/db"
	"github.com/fizza/fizza/internal/httpapi"
	"github.com/spf13/cobra"
)

func newServeCmd(rf *rootFlags) *cobra.Command {
	var (
		host         string
		port         int
		addr         string
		readTimeout  time.Duration
		writeTimeout time.Duration
	)
	c := &cobra.Command{
		Use:   "serve",
		Short: "Open the kanban web UI (local HTTP server)",
		Long: `Start a local web UI for browsing and editing boards.

Opens on http://127.0.0.1:6500 by default. The same SQLite database as the
CLI and MCP server is used. JSON endpoints under /v1 power the UI.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			dbPath, err := config.DBPath()
			if err != nil {
				return report(cmd, rf, err)
			}
			pool, err := db.OpenPool(ctx, dbPath, 4, 1)
			if err != nil {
				return report(cmd, rf, err)
			}
			defer pool.Close()
			srv := httpapi.New(pool.Write, rf.conf.Project)
			listen := addr
			if listen == "" {
				listen = fmt.Sprintf("%s:%d", host, port)
			}
			fmt.Fprintf(cmd.ErrOrStderr(), "fizza web UI on http://%s\n", listen)
			return srv.Run(ctx, httpapi.Options{
				Addr:         listen,
				ReadTimeout:  readTimeout,
				WriteTimeout: writeTimeout,
			})
		},
	}
	c.Flags().StringVar(&host, "host", "127.0.0.1", "Bind host (ignored if --addr is set)")
	c.Flags().IntVar(&port, "port", 6500, "Bind port (ignored if --addr is set)")
	c.Flags().StringVar(&addr, "addr", "", "Full listen address, e.g. 0.0.0.0:9090")
	c.Flags().DurationVar(&readTimeout, "read-timeout", 30*time.Second, "HTTP read timeout")
	c.Flags().DurationVar(&writeTimeout, "write-timeout", 30*time.Second, "HTTP write timeout")
	return c
}
