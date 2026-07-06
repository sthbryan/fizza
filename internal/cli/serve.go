package cli

import (
	"fmt"
	"strconv"
	"strings"
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
		Short: "Run fizza as an HTTP server (REST API over the service layer)",
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
			fmt.Fprintf(cmd.ErrOrStderr(), "fizza serve listening on %s\n", listen)
			return srv.Run(ctx, httpapi.Options{
				Addr:         listen,
				ReadTimeout:  readTimeout,
				WriteTimeout: writeTimeout,
			})
		},
	}
	c.Flags().StringVar(&host, "host", "127.0.0.1", "Bind host (ignored if --addr is set)")
	c.Flags().IntVar(&port, "port", 8080, "Bind port (ignored if --addr is set)")
	c.Flags().StringVar(&addr, "addr", "", "Full listen address, e.g. 0.0.0.0:9090")
	c.Flags().DurationVar(&readTimeout, "read-timeout", 30*time.Second, "HTTP read timeout")
	c.Flags().DurationVar(&writeTimeout, "write-timeout", 30*time.Second, "HTTP write timeout")
	return c
}

func splitHostPort(addr string) (string, int, error) {
	idx := strings.LastIndex(addr, ":")
	if idx < 0 {
		return addr, 0, nil
	}
	port, err := strconv.Atoi(addr[idx+1:])
	if err != nil {
		return "", 0, err
	}
	return addr[:idx], port, nil
}