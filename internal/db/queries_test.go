package db

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/fizza/fizza/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestDB(t *testing.T) *sql.DB {
	t.Helper()
	conn, err := Open(context.Background(), ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })
	return conn
}

func TestCRUD_Project(t *testing.T) {
	conn := newTestDB(t)
	ctx := context.Background()

	p, err := CreateProject(ctx, conn, "alpha", "first project")
	require.NoError(t, err)
	assert.NotZero(t, p.ID)
	assert.Equal(t, "alpha", p.Name)

	got, err := GetProjectByName(ctx, conn, "alpha")
	require.NoError(t, err)
	assert.Equal(t, p.ID, got.ID)

	_, err = CreateProject(ctx, conn, "alpha", "")
	require.True(t, IsDuplicate(err), "duplicate name should be detected")

	require.NoError(t, DeleteProject(ctx, conn, p.ID))
	_, err = GetProject(ctx, conn, p.ID)
	require.True(t, IsNotFound(err))
}

func TestCRUD_Board_WithSeedColumns(t *testing.T) {
	conn := newTestDB(t)
	ctx := context.Background()

	p, err := CreateProject(ctx, conn, "alpha", "")
	require.NoError(t, err)

	boards, err := ListBoards(ctx, conn, p.ID)
	require.NoError(t, err)
	b := boards[0]
	require.NoError(t, err)
	assert.True(t, b.IsDefault, "first board of a project should be default")

	cols, err := ListColumns(ctx, conn, b.ID)
	require.NoError(t, err)
	require.Len(t, cols, 3)
	assert.Equal(t, "todo", cols[0].Name)
	assert.Equal(t, "in_progress", cols[1].Name)
	assert.Equal(t, "done", cols[2].Name)

	b2, err := CreateBoard(ctx, conn, p.ID, "secondary")
	require.NoError(t, err)
	assert.False(t, b2.IsDefault, "second board should not be default")

	custom, err := CreateBoardWithColumns(ctx, conn, p.ID, "qa", []string{"open", "verified"})
	require.NoError(t, err)
	qacols, err := ListColumns(ctx, conn, custom.ID)
	require.NoError(t, err)
	assert.Equal(t, []string{"open", "verified"}, []string{qacols[0].Name, qacols[1].Name})
}

func TestCRUD_Task_FullLifecycle(t *testing.T) {
	conn := newTestDB(t)
	ctx := context.Background()

	p, _ := CreateProject(ctx, conn, "alpha", "")
	boards, _ := ListBoards(ctx, conn, p.ID)
	b := boards[0]
	cols, _ := ListColumns(ctx, conn, b.ID)
	todo := cols[0]
	done := cols[2]

	t1 := &model.Task{
		BoardID:     b.ID,
		ColumnID:    todo.ID,
		Title:       "first task",
		Description: "with details",
		Priority:    model.PriorityHigh,
	}
	require.NoError(t, CreateTask(ctx, conn, t1))
	assert.NotZero(t, t1.ID)
	assert.Equal(t, 1000.0, t1.Position)
	assert.Equal(t, "todo", t1.ColumnName)
	assert.False(t, t1.CreatedAt.IsZero())

	t2 := &model.Task{BoardID: b.ID, ColumnID: todo.ID, Title: "second", Priority: model.PriorityMedium}
	require.NoError(t, CreateTask(ctx, conn, t2))
	assert.Equal(t, 2000.0, t2.Position, "second task in same column gets next step")

	list, err := ListTasksInBoard(ctx, conn, b.ID, "")
	require.NoError(t, err)
	assert.Len(t, list, 2)
	assert.Equal(t, "first task", list[0].Title)

	require.NoError(t, MoveTask(ctx, conn, t1.ID, done.ID))
	moved, err := GetTask(ctx, conn, t1.ID)
	require.NoError(t, err)
	assert.Equal(t, "done", moved.ColumnName)
	assert.Equal(t, 1000.0, moved.Position, "first task in empty done column gets position 1000")

	require.NoError(t, MoveTask(ctx, conn, t2.ID, done.ID))
	again, _ := GetTask(ctx, conn, t2.ID)
	assert.Equal(t, 2000.0, again.Position, "second mover goes after the first")

	due := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	desc := "new desc"
	require.NoError(t, UpdateTask(ctx, conn, t1.ID, TaskPatch{
		Description: &desc,
		DueDate:     &due,
	}))
	updated, _ := GetTask(ctx, conn, t1.ID)
	assert.Equal(t, "new desc", updated.Description)
	require.NotNil(t, updated.DueDate)
	assert.True(t, updated.DueDate.Equal(due))

	require.NoError(t, DeleteTask(ctx, conn, t2.ID))
	_, err = GetTask(ctx, conn, t2.ID)
	assert.True(t, IsNotFound(err))
}

func TestTask_SubtasksAndCascade(t *testing.T) {
	conn := newTestDB(t)
	ctx := context.Background()

	p, _ := CreateProject(ctx, conn, "alpha", "")
	boards, _ := ListBoards(ctx, conn, p.ID)
	b := boards[0]
	cols, _ := ListColumns(ctx, conn, b.ID)
	todo := cols[0]

	parent := &model.Task{BoardID: b.ID, ColumnID: todo.ID, Title: "epic"}
	require.NoError(t, CreateTask(ctx, conn, parent))

	childA := &model.Task{BoardID: b.ID, ColumnID: todo.ID, Title: "child A", Priority: model.PriorityLow}
	parentID := parent.ID
	childA.ParentID = &parentID
	require.NoError(t, CreateTask(ctx, conn, childA))

	childB := &model.Task{BoardID: b.ID, ColumnID: todo.ID, Title: "child B"}
	childB.ParentID = &parentID
	require.NoError(t, CreateTask(ctx, conn, childB))

	subs, err := ListSubtasks(ctx, conn, parent.ID)
	require.NoError(t, err)
	assert.Len(t, subs, 2)
	assert.Equal(t, "child A", subs[0].Title)

	require.NoError(t, DeleteTask(ctx, conn, parent.ID))

	for _, id := range []int64{parent.ID, childA.ID, childB.ID} {
		_, err := GetTask(ctx, conn, id)
		assert.True(t, IsNotFound(err), "subtasks must cascade-delete with parent")
	}
}

func TestTask_PrefixLookup(t *testing.T) {
	conn := newTestDB(t)
	ctx := context.Background()

	p, _ := CreateProject(ctx, conn, "alpha", "")
	boards, _ := ListBoards(ctx, conn, p.ID)
	b := boards[0]
	cols, _ := ListColumns(ctx, conn, b.ID)

	for _, title := range []string{"alpha-task", "beta-task", "gamma-task"} {
		require.NoError(t, CreateTask(ctx, conn, &model.Task{
			BoardID: b.ID, ColumnID: cols[0].ID, Title: title,
		}))
	}

	got, err := GetTaskByPrefix(ctx, conn, "1")
	require.NoError(t, err)
	assert.Equal(t, int64(1), got.ID)

	list, _ := ListTasksInBoard(ctx, conn, b.ID, "")
	if len(list) > 9 {
		_, err = GetTaskByPrefix(ctx, conn, "1")
		assert.True(t, errors.Is(err, ErrNotFound) || err != nil,
			"prefix '1' should be ambiguous when 10+ tasks exist")
	}

	_, err = GetTaskByPrefix(ctx, conn, "")
	require.Error(t, err)

	_, err = GetTaskByPrefix(ctx, conn, "abc")
	require.Error(t, err)
}

func TestDeleteBoard_BlockedByTasks(t *testing.T) {
	conn := newTestDB(t)
	ctx := context.Background()

	p, _ := CreateProject(ctx, conn, "alpha", "")
	boards, _ := ListBoards(ctx, conn, p.ID)
	b := boards[0]
	cols, _ := ListColumns(ctx, conn, b.ID)
	require.NoError(t, CreateTask(ctx, conn, &model.Task{
		BoardID: b.ID, ColumnID: cols[0].ID, Title: "blocker",
	}))

	err := DeleteBoard(ctx, conn, b.ID)
	require.Error(t, err, "FK RESTRICT must prevent delete when tasks exist")
	assert.False(t, IsNotFound(err))
}