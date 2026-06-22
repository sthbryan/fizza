package mcp

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestSession(t *testing.T) *mcp.ClientSession {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "v0"}, nil)
	session, err := client.Connect(ctx, &mcp.CommandTransport{
		Command: commandForServer(dbPath),
	}, nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = session.Close() })
	return session
}

func TestServer_Initialize(t *testing.T) {
	session := newTestSession(t)
	res := session.InitializeResult()
	require.NotNil(t, res)
	assert.Equal(t, "fizza", res.ServerInfo.Name)
}

func TestServer_ToolsList(t *testing.T) {
	session := newTestSession(t)
	tools, err := session.ListTools(context.Background(), &mcp.ListToolsParams{})
	require.NoError(t, err)

	names := map[string]bool{}
	for _, tool := range tools.Tools {
		names[tool.Name] = true
		assert.NotEmpty(t, tool.Description, "tool %s must have a description", tool.Name)
	}

	expected := []string{
		"project_new", "project_list", "project_show", "project_delete",
		"board_create", "board_list", "board_show", "board_delete",
		"task_add", "task_list", "task_show", "task_move", "task_update", "task_delete",
	}
	for _, name := range expected {
		assert.True(t, names[name], "expected tool %q in registry", name)
	}
}

func TestServer_ToolSchemasParseAsJSONSchema(t *testing.T) {
	session := newTestSession(t)
	tools, err := session.ListTools(context.Background(), &mcp.ListToolsParams{})
	require.NoError(t, err)

	for _, tool := range tools.Tools {
		if tool.InputSchema == nil {
			continue
		}
		raw, err := json.Marshal(tool.InputSchema)
		require.NoError(t, err, "tool %s schema must be JSON-encodable", tool.Name)
		var parsed map[string]any
		require.NoError(t, json.Unmarshal(raw, &parsed), "tool %s schema must round-trip JSON", tool.Name)
	}
}

func TestServer_EndToEnd_CreateAndList(t *testing.T) {
	session := newTestSession(t)
	ctx := context.Background()

	res, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "project_new",
		Arguments: map[string]any{
			"name":        "alpha",
			"description": "test project",
		},
	})
	require.NoError(t, err)
	assert.False(t, res.IsError, "project_new should succeed")

	res, err = session.CallTool(ctx, &mcp.CallToolParams{
		Name: "project_list",
		Arguments: map[string]any{},
	})
	require.NoError(t, err)
	assert.False(t, res.IsError)
	assert.NotEmpty(t, res.Content)

	res, err = session.CallTool(ctx, &mcp.CallToolParams{
		Name: "task_add",
		Arguments: map[string]any{
			"project":  "alpha",
			"board":    "main",
			"title":    "first task",
			"priority": "high",
		},
	})
	require.NoError(t, err)
	assert.False(t, res.IsError, "task_add should succeed")

	res, err = session.CallTool(ctx, &mcp.CallToolParams{
		Name: "task_list",
		Arguments: map[string]any{
			"project": "alpha",
			"board":   "main",
		},
	})
	require.NoError(t, err)
	assert.False(t, res.IsError)
}