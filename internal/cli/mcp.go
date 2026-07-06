package cli

import (
	"context"

	fizzamcp "github.com/fizza/fizza/internal/mcp"
	"github.com/spf13/cobra"
)

func newMCPCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "mcp",
		Short: "Run as an MCP server over stdio",
		Long: "Starts a Model Context Protocol server speaking JSON-RPC 2.0 over stdin/stdout. " +
			"Intended to be invoked by an MCP-compatible coding agent (Claude Code, Cursor, etc.).",
		RunE: func(cmd *cobra.Command, args []string) error {
			return fizzamcp.Run(context.Background(), version)
		},
	}
}
