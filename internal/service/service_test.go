package service

import (
	"context"
	"strings"
	"testing"

	"github.com/fizza/fizza/internal/db"
	"github.com/fizza/fizza/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestService(t *testing.T) *Service {
	t.Helper()
	conn, err := db.Open(context.Background(), ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })
	ctx := context.Background()
	_, err = db.CreateProject(ctx, conn, "alpha", "")
	require.NoError(t, err)
	return New(conn, "alpha", "main", "")
}

func TestService_ResolveBoard(t *testing.T) {
	s := newTestService(t)
	r, err := s.ResolveBoard(context.Background())
	require.NoError(t, err)
	require.NotNil(t, r.Project)
	require.NotNil(t, r.Board)
	assert.Equal(t, "alpha", r.Project.Name)
	assert.Equal(t, "main", r.Board.Name)
}

func TestService_ResolveColumn_DefaultFirst(t *testing.T) {
	s := newTestService(t)
	r, err := s.ResolveColumn(context.Background(), true)
	require.NoError(t, err)
	require.NotNil(t, r.Column)
	assert.Equal(t, "todo", r.Column.Name)
}

func TestService_CreateTask(t *testing.T) {
	s := newTestService(t)
	ctx := context.Background()
	task, err := s.CreateTask(ctx, TaskCreateInput{Title: "ship it"})
	require.NoError(t, err)
	assert.NotZero(t, task.ID)
	assert.Equal(t, "ship it", task.Title)
	assert.Equal(t, "todo", task.ColumnName)
}

func TestService_ProjectCounts(t *testing.T) {
	s := newTestService(t)
	ctx := context.Background()
	_, err := s.CreateTask(ctx, TaskCreateInput{Title: "a"})
	require.NoError(t, err)
	c, err := s.ProjectCounts(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(1), c.Projects)
	assert.Equal(t, int64(1), c.Boards)
	assert.Equal(t, int64(1), c.Tasks)
}

func TestSplitColumns(t *testing.T) {
	cases := map[string][]string{
		"":                  nil,
		"todo":              {"todo"},
		"todo,in_progress":  {"todo", "in_progress"},
		" todo , done ":     {"todo", "done"},
		"a,,b":              {"a", "b"},
	}
	for in, want := range cases {
		got := SplitColumns(in)
		if want == nil {
			assert.Empty(t, got, in)
			continue
		}
		assert.Equal(t, want, got, in)
	}
}

func TestParseInt64Flexible(t *testing.T) {
	v, err := ParseInt64Flexible("42")
	require.NoError(t, err)
	assert.Equal(t, int64(42), v)

	_, err = ParseInt64Flexible("")
	assert.Error(t, err)

	_, err = ParseInt64Flexible("1abc")
	assert.True(t, strings.Contains(err.Error(), "not numeric"))
}

func TestPriorityDefault(t *testing.T) {
	s := newTestService(t)
	ctx := context.Background()
	task, err := s.CreateTask(ctx, TaskCreateInput{Title: "x"})
	require.NoError(t, err)
	assert.Equal(t, model.DefaultPriority, task.Priority.String())
}