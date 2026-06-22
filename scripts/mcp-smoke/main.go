// scripts/mcp-smoke/main.go
// Smoke-test the fizza MCP server. Spawns the binary, runs an end-to-end
// flow using the official MCP Go SDK, and prints PASS/FAIL.
package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func main() {
	bin := "./fizza"
	if len(os.Args) > 1 {
		bin = os.Args[1]
	}
	abs, err := filepath.Abs(bin)
	if err != nil {
		die("abs: %v", err)
	}

	dbPath := filepath.Join(os.TempDir(), fmt.Sprintf("fizza-smoke-%d.db", time.Now().UnixNano()))
	defer os.Remove(dbPath)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client := mcp.NewClient(&mcp.Implementation{Name: "fizza-smoke", Version: "0"}, nil)
	session, err := client.Connect(ctx, &mcp.CommandTransport{
		Command: exec.Command(abs, "mcp", "--db", dbPath),
	}, nil)
	if err != nil {
		die("connect: %v", err)
	}
	defer session.Close()

	step("initialize", func() error {
		res := session.InitializeResult()
		if res.ServerInfo.Name != "fizza" {
			return fmt.Errorf("server name = %q, want fizza", res.ServerInfo.Name)
		}
		return nil
	})

	step("tools/list has 14", func() error {
		tools, err := session.ListTools(ctx, &mcp.ListToolsParams{})
		if err != nil {
			return err
		}
		if len(tools.Tools) != 14 {
			return fmt.Errorf("got %d tools, want 14", len(tools.Tools))
		}
		return nil
	})

	step("project_new", func() error {
		_, err := session.CallTool(ctx, &mcp.CallToolParams{
			Name:      "project_new",
			Arguments: map[string]any{"name": "smoke"},
		})
		return err
	})

	step("task_add", func() error {
		_, err := session.CallTool(ctx, &mcp.CallToolParams{
			Name: "task_add",
			Arguments: map[string]any{
				"project": "smoke",
				"board":   "main",
				"title":   "hello from mcp",
			},
		})
		return err
	})

	step("task_list returns 1", func() error {
		res, err := session.CallTool(ctx, &mcp.CallToolParams{
			Name: "task_list",
			Arguments: map[string]any{
				"project": "smoke",
				"board":   "main",
			},
		})
		if err != nil {
			return err
		}
		if res.IsError {
			return fmt.Errorf("tool returned error: %+v", res.Content)
		}
		return nil
	})

	step("task_move", func() error {
		_, err := session.CallTool(ctx, &mcp.CallToolParams{
			Name: "task_move",
			Arguments: map[string]any{
				"id":      "1",
				"project": "smoke",
				"board":   "main",
				"column":  "done",
			},
		})
		return err
	})

	fmt.Println("\nall smoke checks passed")
}

func step(label string, fn func() error) {
	fmt.Printf("--- %s\n", label)
	if err := fn(); err != nil {
		die("FAIL: %v", err)
	}
	fmt.Println("OK")
}

func die(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
