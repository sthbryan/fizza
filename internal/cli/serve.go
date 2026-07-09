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
		noOpen       bool
	)
	c := &cobra.Command{
		Use:   "serve",
		Short: "Open the kanban web UI (local HTTP server)",
		Long: `Start a local web UI for browsing and editing boards.

Opens on http://127.0.0.1:6500 by default and launches the system browser.
The same SQLite database as the CLI and MCP server is used. JSON endpoints
under /v1 power the UI. Pass --no-open to only print the URL.`,
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
			url := browserURL(listen)
			fmt.Fprintf(cmd.ErrOrStderr(), "fizza web UI on %s\n", url)
			if !noOpen {
				go func() {
					time.Sleep(150 * time.Millisecond)
					if err := openBrowser(url); err != nil {
						fmt.Fprintf(cmd.ErrOrStderr(), "could not open browser: %v\n", err)
					}
				}()
			}
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
	c.Flags().DurationVar(&writeTimeout, "write-timeout", 0, "HTTP write timeout (0 = none; needed for SSE)")
	c.Flags().BoolVar(&noOpen, "no-open", false, "Do not open the system browser")
	return c
}
