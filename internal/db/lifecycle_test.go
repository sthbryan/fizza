package db

import (
	"context"
	"testing"

	"github.com/fizza/fizza/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTaskLifecycle_CompletedAndArchive(t *testing.T) {
	conn := newTestDB(t)
	ctx := context.Background()

	p, err := CreateProject(ctx, conn, "alpha", "")
	require.NoError(t, err)
	boards, err := ListBoards(ctx, conn, p.ID)
	require.NoError(t, err)
	board := boards[0]
	cols, err := ListColumns(ctx, conn, board.ID)
	require.NoError(t, err)
	todo := cols[0]
	done := cols[len(cols)-1]
	require.Equal(t, "done", done.Name)

	task := &model.Task{BoardID: board.ID, ColumnID: todo.ID, Title: "work"}
	require.NoError(t, CreateTask(ctx, conn, task))
	require.Nil(t, task.CompletedAt)
	require.Nil(t, task.ArchivedAt)

	require.NoError(t, MoveTask(ctx, conn, task.ID, done.ID))
	got, err := GetTask(ctx, conn, task.ID)
	require.NoError(t, err)
	require.NotNil(t, got.CompletedAt)
	assert.Equal(t, "done", got.ColumnName)

	require.NoError(t, MoveTask(ctx, conn, task.ID, todo.ID))
	got, err = GetTask(ctx, conn, task.ID)
	require.NoError(t, err)
	assert.Nil(t, got.CompletedAt)

	require.NoError(t, MoveTask(ctx, conn, task.ID, done.ID))
	require.NoError(t, ArchiveTask(ctx, conn, task.ID))
	got, err = GetTask(ctx, conn, task.ID)
	require.NoError(t, err)
	require.NotNil(t, got.ArchivedAt)

	active, err := ListTasksInBoard(ctx, conn, board.ID, TaskFilter{IncludeDone: true})
	require.NoError(t, err)
	assert.Empty(t, active)

	archived, err := ListTasksInBoard(ctx, conn, board.ID, TaskFilter{OnlyArchived: true})
	require.NoError(t, err)
	require.Len(t, archived, 1)
	assert.Equal(t, task.ID, archived[0].ID)

	err = MoveTask(ctx, conn, task.ID, todo.ID)
	require.Error(t, err)

	require.NoError(t, UnarchiveTask(ctx, conn, task.ID))
	got, err = GetTask(ctx, conn, task.ID)
	require.NoError(t, err)
	assert.Nil(t, got.ArchivedAt)
	assert.Equal(t, "done", got.ColumnName)

	n, err := ArchiveDoneInBoard(ctx, conn, board.ID)
	require.NoError(t, err)
	assert.Equal(t, int64(1), n)

	count, err := CountArchivedInBoard(ctx, conn, board.ID)
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)
}

func TestListTasks_ExcludeDoneByDefault(t *testing.T) {
	conn := newTestDB(t)
	ctx := context.Background()
	p, _ := CreateProject(ctx, conn, "alpha", "")
	boards, _ := ListBoards(ctx, conn, p.ID)
	board := boards[0]
	cols, _ := ListColumns(ctx, conn, board.ID)
	todo, done := cols[0], cols[len(cols)-1]

	require.NoError(t, CreateTask(ctx, conn, &model.Task{BoardID: board.ID, ColumnID: todo.ID, Title: "open"}))
	require.NoError(t, CreateTask(ctx, conn, &model.Task{BoardID: board.ID, ColumnID: done.ID, Title: "closed"}))

	openOnly, err := ListTasksInBoard(ctx, conn, board.ID, TaskFilter{})
	require.NoError(t, err)
	require.Len(t, openOnly, 1)
	assert.Equal(t, "open", openOnly[0].Title)

	all, err := ListTasksInBoard(ctx, conn, board.ID, TaskFilter{IncludeDone: true})
	require.NoError(t, err)
	assert.Len(t, all, 2)

	doneOnly, err := ListTasksInBoard(ctx, conn, board.ID, TaskFilter{ColumnName: "done"})
	require.NoError(t, err)
	require.Len(t, doneOnly, 1)
	assert.Equal(t, "closed", doneOnly[0].Title)
}
