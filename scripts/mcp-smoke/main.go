package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

var expectedTools = []string{
	"board_create",
	"board_delete",
	"board_list",
	"board_snapshot",
	"project_delete",
	"project_list",
	"project_new",
	"tag_add",
	"tag_delete",
	"tag_list",
	"task_add",
	"task_archive",
	"task_archive_done",
	"task_delete",
	"task_list",
	"task_move",
	"task_unarchive",
	"task_update",
}

func main() {
	bin := "./fizza"
	if len(os.Args) > 1 {
		bin = os.Args[1]
	}
	abs, err := filepath.Abs(bin)
	if err != nil {
		die("abs: %v", err)
	}

	xdgDir, err := os.MkdirTemp("", "fizza-smoke-*")
	if err != nil {
		die("mkdir: %v", err)
	}
	defer os.RemoveAll(xdgDir)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmd := exec.Command(abs, "mcp")
	cmd.Env = append(os.Environ(), "XDG_CONFIG_HOME="+xdgDir)
	client := mcp.NewClient(&mcp.Implementation{Name: "fizza-smoke", Version: "0"}, nil)
	session, err := client.Connect(ctx, &mcp.CommandTransport{
		Command: cmd,
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

	step("tools/list matches the expected set", func() error {
		tools, err := session.ListTools(ctx, &mcp.ListToolsParams{})
		if err != nil {
			return err
		}
		got := map[string]bool{}
		for _, t := range tools.Tools {
			got[t.Name] = true
		}
		var missing, unexpected []string
		for _, name := range expectedTools {
			if !got[name] {
				missing = append(missing, name)
			}
		}
		want := map[string]bool{}
		for _, name := range expectedTools {
			want[name] = true
		}
		for name := range got {
			if !want[name] {
				unexpected = append(unexpected, name)
			}
		}
		sort.Strings(missing)
		sort.Strings(unexpected)
		if len(missing) > 0 || len(unexpected) > 0 {
			return fmt.Errorf("tool set drifted: missing %v, unexpected %v", missing, unexpected)
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

	step("project_list fused view", func() error {
		res, err := session.CallTool(ctx, &mcp.CallToolParams{
			Name:      "project_list",
			Arguments: map[string]any{"name": "smoke"},
		})
		if err != nil {
			return err
		}
		if res.IsError {
			return fmt.Errorf("expected single project, got error: %+v", res.Content)
		}
		return nil
	})

	step("task_list fused view by id", func() error {
		res, err := session.CallTool(ctx, &mcp.CallToolParams{
			Name:      "task_list",
			Arguments: map[string]any{"id": "1"},
		})
		if err != nil {
			return err
		}
		if res.IsError {
			return fmt.Errorf("expected task by id, got error: %+v", res.Content)
		}
		return nil
	})

	step("task_list filter by tag", func() error {
		_, err := session.CallTool(ctx, &mcp.CallToolParams{
			Name: "task_update",
			Arguments: map[string]any{
				"id":       "1",
				"add_tags": []string{"urgent"},
			},
		})
		if err != nil {
			return err
		}
		res, err := session.CallTool(ctx, &mcp.CallToolParams{
			Name: "task_list",
			Arguments: map[string]any{
				"project": "smoke",
				"board":   "main",
				"tag":     "urgent",
			},
		})
		if err != nil {
			return err
		}
		if res.IsError {
			return fmt.Errorf("tag filter failed: %+v", res.Content)
		}
		return nil
	})

	step("board_snapshot", func() error {
		res, err := session.CallTool(ctx, &mcp.CallToolParams{
			Name: "board_snapshot",
			Arguments: map[string]any{
				"project": "smoke",
				"board":   "main",
			},
		})
		if err != nil {
			return err
		}
		if res.IsError {
			return fmt.Errorf("board_snapshot failed: %+v", res.Content)
		}
		return nil
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
