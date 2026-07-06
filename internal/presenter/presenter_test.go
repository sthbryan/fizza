package presenter

import (
	"bytes"
	"testing"
	"time"

	"github.com/fizza/fizza/internal/config"
	"github.com/fizza/fizza/internal/dbutil"
	"github.com/fizza/fizza/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProject(t *testing.T) {
	var buf bytes.Buffer
	r := New(&buf, true)
	now := time.Date(2026, 7, 5, 12, 0, 0, 0, time.UTC)
	require.NoError(t, r.Project(&model.Project{ID: 1, Name: "alpha", Description: "d", CreatedAt: dbutil.Time{Time: now}, UpdatedAt: dbutil.Time{Time: now}}))
	out := buf.String()
	assert.Contains(t, out, "ID:")
	assert.Contains(t, out, "alpha")
	assert.Contains(t, out, "d")
}

func TestProjectList_Empty(t *testing.T) {
	var buf bytes.Buffer
	r := New(&buf, true)
	require.NoError(t, r.ProjectList(nil))
	assert.Equal(t, "no projects\n", buf.String())
}

func TestBoard(t *testing.T) {
	var buf bytes.Buffer
	r := New(&buf, true)
	now := time.Date(2026, 7, 5, 12, 0, 0, 0, time.UTC)
	require.NoError(t, r.Board(&model.Board{ID: 2, ProjectID: 1, Name: "main", IsDefault: true, CreatedAt: dbutil.Time{Time: now}}))
	out := buf.String()
	assert.Contains(t, out, "yes")
	assert.Contains(t, out, "main")
}

func TestTask(t *testing.T) {
	var buf bytes.Buffer
	r := New(&buf, true)
	due := time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC)
	parent := int64(7)
	require.NoError(t, r.Task(&model.Task{
		ID: 5, BoardID: 1, ColumnID: 2, ColumnName: "todo",
		Title: "ship it", Priority: model.Priority{Value: model.PriorityHigh},
		ParentID: &parent, DueDate: &dbutil.Time{Time: due},
	}))
	out := buf.String()
	assert.Contains(t, out, "todo")
	assert.Contains(t, out, "ship it")
	assert.Contains(t, out, "high")
	assert.Contains(t, out, "7")
	assert.Contains(t, out, "2026-12-01")
}

func TestConfig(t *testing.T) {
	var buf bytes.Buffer
	r := New(&buf, true)
	require.NoError(t, r.Config(config.Config{Mode: "llm", Project: "alpha"}))
	out := buf.String()
	assert.Contains(t, out, "llm")
	assert.Contains(t, out, "alpha")
}

func TestConfig_UnsetProject(t *testing.T) {
	var buf bytes.Buffer
	r := New(&buf, true)
	require.NoError(t, r.Config(config.Config{Mode: "llm"}))
	out := buf.String()
	assert.Contains(t, out, "(unset)")
}

func TestTaskList_Empty(t *testing.T) {
	var buf bytes.Buffer
	r := New(&buf, true)
	require.NoError(t, r.TaskList(nil))
	assert.Equal(t, "no tasks\n", buf.String())
}
