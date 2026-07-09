package db

import (
	"context"
	"testing"
	"time"

	"github.com/fizza/fizza/internal/dbutil"
	"github.com/fizza/fizza/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetStats_Empty(t *testing.T) {
	conn := newTestDB(t)
	ctx := context.Background()

	stats, err := GetStats(ctx, conn, "", "")
	require.NoError(t, err)
	assert.Equal(t, int64(0), stats.Totals.Projects)
	assert.Equal(t, int64(0), stats.Totals.Tasks)
	assert.Len(t, stats.ByPriority, 4)
	assert.Empty(t, stats.ByProject)
	assert.Empty(t, stats.ByBoard)
}

func TestGetStats_WithTasks(t *testing.T) {
	conn := newTestDB(t)
	ctx := context.Background()

	p, err := CreateProject(ctx, conn, "alpha", "")
	require.NoError(t, err)
	boards, err := ListBoards(ctx, conn, p.ID)
	require.NoError(t, err)
	require.NotEmpty(t, boards)
	board := boards[0]
	cols, err := ListColumns(ctx, conn, board.ID)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(cols), 2)

	todo := cols[0]
	doneCol := cols[len(cols)-1]
	require.Equal(t, "done", doneCol.Name)

	t1 := &model.Task{
		BoardID:  board.ID,
		ColumnID: todo.ID,
		Title:    "open work",
		Priority: model.MustPriority("high"),
	}
	require.NoError(t, CreateTask(ctx, conn, t1))

	past := dbutil.Time{Time: time.Now().UTC().Add(-48 * time.Hour)}
	t2 := &model.Task{
		BoardID:  board.ID,
		ColumnID: todo.ID,
		Title:    "overdue work",
		Priority: model.MustPriority("urgent"),
		DueDate:  &past,
	}
	require.NoError(t, CreateTask(ctx, conn, t2))

	t3 := &model.Task{
		BoardID:  board.ID,
		ColumnID: doneCol.ID,
		Title:    "finished",
		Priority: model.MustPriority("low"),
	}
	require.NoError(t, CreateTask(ctx, conn, t3))

	stats, err := GetStats(ctx, conn, "", "")
	require.NoError(t, err)
	assert.Equal(t, int64(1), stats.Totals.Projects)
	assert.Equal(t, int64(1), stats.Totals.Boards)
	assert.Equal(t, int64(3), stats.Totals.Tasks)
	assert.Equal(t, int64(1), stats.Totals.Done)
	assert.Equal(t, int64(2), stats.Totals.Open)
	assert.Equal(t, int64(1), stats.Totals.Overdue)
	assert.Equal(t, int64(0), stats.Totals.Archived)

	require.Len(t, stats.ByPriority, 4)
	pri := map[string]int64{}
	for _, r := range stats.ByPriority {
		pri[r.Name] = r.Count
	}
	assert.Equal(t, int64(1), pri["high"])
	assert.Equal(t, int64(1), pri["urgent"])
	assert.Equal(t, int64(1), pri["low"])
	assert.Equal(t, int64(0), pri["medium"])

	scoped, err := GetStats(ctx, conn, "alpha", "")
	require.NoError(t, err)
	assert.Equal(t, "alpha", scoped.Scope.Project)
	assert.Equal(t, int64(3), scoped.Totals.Tasks)
	assert.Empty(t, scoped.ByProject)
	require.NotEmpty(t, scoped.ByBoard)

	boardScoped, err := GetStats(ctx, conn, "alpha", board.Name)
	require.NoError(t, err)
	assert.Equal(t, board.Name, boardScoped.Scope.Board)
	assert.Equal(t, int64(3), boardScoped.Totals.Tasks)
	assert.Empty(t, boardScoped.ByBoard)

	_, err = GetStats(ctx, conn, "nope", "")
	require.Error(t, err)
	assert.True(t, IsNotFound(err))
}
