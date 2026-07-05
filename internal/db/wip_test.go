package db

import (
	"context"
	"testing"

	"github.com/fizza/fizza/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func seedWIPFixture(t *testing.T) (board *model.Board, todo, inProgress, done *model.Column) {
	t.Helper()
	conn := newTestDB(t)
	ctx := context.Background()
	p, err := CreateProject(ctx, conn, "alpha", "")
	require.NoError(t, err)
	boards, err := ListBoards(ctx, conn, p.ID)
	require.NoError(t, err)
	cols, err := ListColumns(ctx, conn, boards[0].ID)
	require.NoError(t, err)
	return boards[0], cols[0], cols[1], cols[2]
}

func TestWIP_ColumnExposesLimit(t *testing.T) {
	conn := newTestDB(t)
	ctx := context.Background()
	_, todo, inProgress, _ := seedWIPFixture(t)

	require.NoError(t, UpdateColumnWIPLimit(ctx, conn, inProgress.ID, intPtr(3)))

	got, err := GetColumnByName(ctx, conn, todo.BoardID, "in_progress")
	require.NoError(t, err)
	require.NotNil(t, got.WIPLimit)
	assert.Equal(t, 3, *got.WIPLimit)

	listed, err := ListColumns(ctx, conn, todo.BoardID)
	require.NoError(t, err)
	var found *model.Column
	for _, c := range listed {
		if c.ID == inProgress.ID {
			found = c
		}
	}
	require.NotNil(t, found)
	require.NotNil(t, found.WIPLimit)
	assert.Equal(t, 3, *found.WIPLimit)

	none, err := GetColumnByName(ctx, conn, todo.BoardID, "todo")
	require.NoError(t, err)
	assert.Nil(t, none.WIPLimit, "columns without a limit must serialize nil")
}

func TestWIP_MoveRejectsOverLimit(t *testing.T) {
	conn := newTestDB(t)
	ctx := context.Background()
	b, todo, inProgress, _ := seedWIPFixture(t)

	limit := 2
	require.NoError(t, UpdateColumnWIPLimit(ctx, conn, inProgress.ID, &limit))

	mk := func(title string) *model.Task {
		t1 := &model.Task{BoardID: b.ID, ColumnID: todo.ID, Title: title}
		require.NoError(t, CreateTask(ctx, conn, t1))
		return t1
	}
	tA := mk("alpha")
	tB := mk("beta")
	tC := mk("gamma")

	require.NoError(t, MoveTask(ctx, conn, tA.ID, inProgress.ID))
	require.NoError(t, MoveTask(ctx, conn, tB.ID, inProgress.ID))

	err := MoveTask(ctx, conn, tC.ID, inProgress.ID)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrWIPLimitReached)

	stillTodo, err := GetTask(ctx, conn, tC.ID)
	require.NoError(t, err)
	assert.Equal(t, todo.ID, stillTodo.ColumnID, "rejected move must not relocate the task")

	list, err := ListTasksInColumn(ctx, conn, inProgress.ID)
	require.NoError(t, err)
	assert.Len(t, list, 2)
}

func TestWIP_MoveForceBypasses(t *testing.T) {
	conn := newTestDB(t)
	ctx := context.Background()
	b, todo, inProgress, _ := seedWIPFixture(t)

	limit := 2
	require.NoError(t, UpdateColumnWIPLimit(ctx, conn, inProgress.ID, &limit))

	mk := func(title string) *model.Task {
		t1 := &model.Task{BoardID: b.ID, ColumnID: todo.ID, Title: title}
		require.NoError(t, CreateTask(ctx, conn, t1))
		return t1
	}
	tA := mk("alpha")
	tB := mk("beta")
	tC := mk("gamma")

	require.NoError(t, MoveTask(ctx, conn, tA.ID, inProgress.ID))
	require.NoError(t, MoveTask(ctx, conn, tB.ID, inProgress.ID))

	require.NoError(t, MoveTaskForce(ctx, conn, tC.ID, inProgress.ID, nil))

	list, err := ListTasksInColumn(ctx, conn, inProgress.ID)
	require.NoError(t, err)
	assert.Len(t, list, 3, "force move must succeed even past the WIP limit")

	got, err := GetTask(ctx, conn, tC.ID)
	require.NoError(t, err)
	assert.Equal(t, inProgress.ID, got.ColumnID)
}

func TestWIP_NullLimitAllowsAnyCount(t *testing.T) {
	conn := newTestDB(t)
	ctx := context.Background()
	b, todo, inProgress, _ := seedWIPFixture(t)

	mk := func(title string) *model.Task {
		t1 := &model.Task{BoardID: b.ID, ColumnID: todo.ID, Title: title}
		require.NoError(t, CreateTask(ctx, conn, t1))
		return t1
	}
	for _, title := range []string{"a", "b", "c", "d", "e"} {
		require.NoError(t, MoveTask(ctx, conn, mk(title).ID, inProgress.ID))
	}
	list, err := ListTasksInColumn(ctx, conn, inProgress.ID)
	require.NoError(t, err)
	assert.Len(t, list, 5)
}

func TestWIP_ClearLimit(t *testing.T) {
	conn := newTestDB(t)
	ctx := context.Background()
	_, _, inProgress, _ := seedWIPFixture(t)

	require.NoError(t, UpdateColumnWIPLimit(ctx, conn, inProgress.ID, intPtr(4)))
	require.NoError(t, UpdateColumnWIPLimit(ctx, conn, inProgress.ID, nil))

	got, err := GetColumnByName(ctx, conn, inProgress.BoardID, "in_progress")
	require.NoError(t, err)
	assert.Nil(t, got.WIPLimit, "clearing the limit must surface nil on read")
}

func intPtr(n int) *int { return &n }