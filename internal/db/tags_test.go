package db

import (
	"context"
	"github.com/jmoiron/sqlx"
	"testing"

	"github.com/fizza/fizza/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func seedTagFixture(t *testing.T, conn *sqlx.DB) (projectID, boardID, colID, taskAID, taskBID int64) {
	t.Helper()
	ctx := context.Background()
	p, err := CreateProject(ctx, conn, "alpha", "")
	require.NoError(t, err)
	boards, err := ListBoards(ctx, conn, p.ID)
	require.NoError(t, err)
	b := boards[0]
	cols, err := ListColumns(ctx, conn, b.ID)
	require.NoError(t, err)

	tA := &model.Task{BoardID: b.ID, ColumnID: cols[0].ID, Title: "alpha-task"}
	require.NoError(t, CreateTask(ctx, conn, tA))
	tB := &model.Task{BoardID: b.ID, ColumnID: cols[0].ID, Title: "beta-task"}
	require.NoError(t, CreateTask(ctx, conn, tB))
	return p.ID, b.ID, cols[0].ID, tA.ID, tB.ID
}

func TestTag_CreateAndList(t *testing.T) {
	conn := newTestDB(t)
	ctx := context.Background()
	pid, _, _, _, _ := seedTagFixture(t, conn)

	urgent, err := CreateTag(ctx, conn, pid, "urgent")
	require.NoError(t, err)
	assert.NotZero(t, urgent.ID)
	assert.Equal(t, "urgent", urgent.Name)
	assert.Equal(t, pid, urgent.ProjectID)
	assert.False(t, urgent.CreatedAt.IsZero())

	_, err = CreateTag(ctx, conn, pid, "bug")
	require.NoError(t, err)

	dup, err := CreateTag(ctx, conn, pid, "urgent")
	require.Error(t, err)
	assert.True(t, IsDuplicate(err), "duplicate tag name within project should be rejected")
	assert.Nil(t, dup)

	got, err := ListTags(ctx, conn, pid)
	require.NoError(t, err)
	require.Len(t, got, 2)
	assert.Equal(t, "bug", got[0].Name, "ListTags must order by name")
	assert.Equal(t, "urgent", got[1].Name)

	other, err := CreateProject(ctx, conn, "beta", "")
	require.NoError(t, err)
	_, err = CreateTag(ctx, conn, other.ID, "urgent")
	require.NoError(t, err, "tag names are scoped per project")

	cross, err := ListTags(ctx, conn, other.ID)
	require.NoError(t, err)
	require.Len(t, cross, 1)
	assert.Equal(t, "urgent", cross[0].Name)
}

func TestTag_Delete(t *testing.T) {
	conn := newTestDB(t)
	ctx := context.Background()
	pid, _, _, _, _ := seedTagFixture(t, conn)

	tag, err := CreateTag(ctx, conn, pid, "scratch")
	require.NoError(t, err)

	require.NoError(t, DeleteTag(ctx, conn, tag.ID))

	got, err := ListTags(ctx, conn, pid)
	require.NoError(t, err)
	assert.Empty(t, got)

	err = DeleteTag(ctx, conn, tag.ID)
	require.Error(t, err)
	assert.True(t, IsNotFound(err), "deleting a missing tag must surface ErrNotFound")

	err = DeleteTag(ctx, conn, 9999)
	require.Error(t, err)
	assert.True(t, IsNotFound(err))
}

func TestTag_AttachDetach(t *testing.T) {
	conn := newTestDB(t)
	ctx := context.Background()
	pid, _, _, taskA, taskB := seedTagFixture(t, conn)

	urgent, err := CreateTag(ctx, conn, pid, "urgent")
	require.NoError(t, err)
	bug, err := CreateTag(ctx, conn, pid, "bug")
	require.NoError(t, err)

	require.NoError(t, AddTagToTask(ctx, conn, taskA, urgent.ID))
	require.NoError(t, AddTagToTask(ctx, conn, taskA, bug.ID))
	require.NoError(t, AddTagToTask(ctx, conn, taskB, urgent.ID))

	require.NoError(t, AddTagToTask(ctx, conn, taskA, urgent.ID), "re-attach is idempotent")

	gotA, err := ListTagsForTask(ctx, conn, taskA)
	require.NoError(t, err)
	require.Len(t, gotA, 2)
	assert.Equal(t, "bug", gotA[0].Name)
	assert.Equal(t, "urgent", gotA[1].Name)

	tasksForUrgent, err := ListTaskIDsForTag(ctx, conn, urgent.ID)
	require.NoError(t, err)
	assert.Equal(t, []int64{taskA, taskB}, tasksForUrgent)

	tasksForBug, err := ListTaskIDsForTag(ctx, conn, bug.ID)
	require.NoError(t, err)
	assert.Equal(t, []int64{taskA}, tasksForBug)

	require.NoError(t, RemoveTagFromTask(ctx, conn, taskA, bug.ID))
	gotA, err = ListTagsForTask(ctx, conn, taskA)
	require.NoError(t, err)
	require.Len(t, gotA, 1)
	assert.Equal(t, "urgent", gotA[0].Name)

	err = RemoveTagFromTask(ctx, conn, taskA, bug.ID)
	require.Error(t, err)
	assert.True(t, IsNotFound(err))
}

func TestTag_CascadeDeleteOnTaskDelete(t *testing.T) {
	conn := newTestDB(t)
	ctx := context.Background()
	pid, _, _, taskA, taskB := seedTagFixture(t, conn)

	urgent, err := CreateTag(ctx, conn, pid, "urgent")
	require.NoError(t, err)
	bug, err := CreateTag(ctx, conn, pid, "bug")
	require.NoError(t, err)

	require.NoError(t, AddTagToTask(ctx, conn, taskA, urgent.ID))
	require.NoError(t, AddTagToTask(ctx, conn, taskA, bug.ID))
	require.NoError(t, AddTagToTask(ctx, conn, taskB, urgent.ID))

	require.NoError(t, DeleteTask(ctx, conn, taskA))

	tasksForUrgent, err := ListTaskIDsForTag(ctx, conn, urgent.ID)
	require.NoError(t, err)
	assert.Equal(t, []int64{taskB}, tasksForUrgent, "task_tags must cascade-delete with the task")

	tasksForBug, err := ListTaskIDsForTag(ctx, conn, bug.ID)
	require.NoError(t, err)
	assert.Empty(t, tasksForBug, "deleting the only attached task must empty the tag's task list")

	tagList, err := ListTags(ctx, conn, pid)
	require.NoError(t, err)
	require.Len(t, tagList, 2, "tags themselves should not be deleted by task deletion")
}

func TestTag_FilterTasksByTagName(t *testing.T) {
	conn := newTestDB(t)
	ctx := context.Background()
	pid, boardID, _, taskA, taskB := seedTagFixture(t, conn)

	urgent, err := CreateTag(ctx, conn, pid, "urgent")
	require.NoError(t, err)
	bug, err := CreateTag(ctx, conn, pid, "bug")
	require.NoError(t, err)

	require.NoError(t, AddTagToTask(ctx, conn, taskA, urgent.ID))
	require.NoError(t, AddTagToTask(ctx, conn, taskA, bug.ID))
	require.NoError(t, AddTagToTask(ctx, conn, taskB, urgent.ID))

	urgentOnly, err := ListTasksInBoard(ctx, conn, boardID, TaskFilter{Tags: []string{"urgent"}})
	require.NoError(t, err)
	require.Len(t, urgentOnly, 2, "filter by 'urgent' returns both tagged tasks")
	idSet := map[int64]string{}
	for _, tk := range urgentOnly {
		idSet[tk.ID] = tk.Title
	}
	assert.Equal(t, "alpha-task", idSet[taskA])
	assert.Equal(t, "beta-task", idSet[taskB])

	bugOnly, err := ListTasksInBoard(ctx, conn, boardID, TaskFilter{Tags: []string{"bug"}})
	require.NoError(t, err)
	require.Len(t, bugOnly, 1)
	assert.Equal(t, taskA, bugOnly[0].ID)

	both, err := ListTasksInBoard(ctx, conn, boardID, TaskFilter{Tags: []string{"urgent", "bug"}})
	require.NoError(t, err)
	require.Len(t, both, 2, "OR filter returns union (task A matches both, task B matches urgent)")
	idSetBool := map[int64]bool{}
	for _, tk := range both {
		idSetBool[tk.ID] = true
	}
	assert.True(t, idSetBool[taskA])
	assert.True(t, idSetBool[taskB])

	missing, err := ListTasksInBoard(ctx, conn, boardID, TaskFilter{Tags: []string{"nonexistent"}})
	require.NoError(t, err)
	assert.Empty(t, missing)

	combined, err := ListTasksInBoard(ctx, conn, boardID, TaskFilter{
		ColumnName: "todo",
		Tags:       []string{"urgent"},
	})
	require.NoError(t, err)
	assert.Len(t, combined, 2, "tag filter must compose with column filter")
}