package db

import (
	"context"
	"errors"
	"sort"
	"sync"
	"testing"

	"github.com/fizza/fizza/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPosition_FractionalInsert(t *testing.T) {
	conn := newTestDB(t)
	ctx := context.Background()

	p, _ := CreateProject(ctx, conn, "p", "")
	boards, _ := ListBoards(ctx, conn, p.ID)
	b := boards[0]
	cols, _ := ListColumns(ctx, conn, b.ID)
	todo := cols[0]

	a := &model.Task{BoardID: b.ID, ColumnID: todo.ID, Title: "A"}
	require.NoError(t, CreateTask(ctx, conn, a))
	require.Equal(t, 1000.0, a.Position)

	b2 := &model.Task{BoardID: b.ID, ColumnID: todo.ID, Title: "B"}
	require.NoError(t, CreateTask(ctx, conn, b2))
	require.Equal(t, 2000.0, b2.Position)

	c := &model.Task{BoardID: b.ID, ColumnID: todo.ID, Title: "C"}
	require.NoError(t, CreateTask(ctx, conn, c))
	require.Equal(t, 3000.0, c.Position)

	mid := &model.Task{BoardID: b.ID, ColumnID: todo.ID, Title: "between A and B"}
	aID := a.ID
	mid.Position, _ = computeNextPosition(ctx, conn, todo.ID, &aID)
	require.NoError(t, CreateTask(ctx, conn, mid))
	assert.Equal(t, 1500.0, mid.Position, "insert between 1000 and 2000 → 1500")

	list, err := ListTasksInColumn(ctx, conn, todo.ID)
	require.NoError(t, err)
	require.Len(t, list, 4)
	assert.Equal(t, []string{"A", "between A and B", "B", "C"},
		[]string{list[0].Title, list[1].Title, list[2].Title, list[3].Title})
}

func TestPosition_AppendAtEnd(t *testing.T) {
	conn := newTestDB(t)
	ctx := context.Background()
	p, _ := CreateProject(ctx, conn, "p", "")
	boards, _ := ListBoards(ctx, conn, p.ID)
	b := boards[0]
	cols, _ := ListColumns(ctx, conn, b.ID)
	todo := cols[0]

	for i := 0; i < 3; i++ {
		tt := &model.Task{BoardID: b.ID, ColumnID: todo.ID, Title: "x"}
		require.NoError(t, CreateTask(ctx, conn, tt))
	}

	positions := []float64{}
	rows, _ := conn.QueryContext(ctx, `SELECT position FROM tasks WHERE column_id=? ORDER BY position`, todo.ID)
	for rows.Next() {
		var p float64
		require.NoError(t, rows.Scan(&p))
		positions = append(positions, p)
	}
	_ = rows.Close()
	assert.Equal(t, []float64{1000, 2000, 3000}, positions)
}

func TestPosition_RebalanceAfterManyInsertsInSameSpot(t *testing.T) {
	conn := newTestDB(t)
	ctx := context.Background()
	p, _ := CreateProject(ctx, conn, "p", "")
	boards, _ := ListBoards(ctx, conn, p.ID)
	b := boards[0]
	cols, _ := ListColumns(ctx, conn, b.ID)
	todo := cols[0]

	a := &model.Task{BoardID: b.ID, ColumnID: todo.ID, Title: "anchor"}
	require.NoError(t, CreateTask(ctx, conn, a))
	b2 := &model.Task{BoardID: b.ID, ColumnID: todo.ID, Title: "anchor2"}
	require.NoError(t, CreateTask(ctx, conn, b2))

	for i := 0; i < 60; i++ {
		bID := a.ID
		pos, err := computeNextPosition(ctx, conn, todo.ID, &bID)
		if errors.Is(err, errGapTooSmall) {
			_, _ = RebalanceColumn(ctx, conn, todo.ID)
			pos, err = computeNextPosition(ctx, conn, todo.ID, &bID)
		}
		require.NoError(t, err)
		tt := &model.Task{BoardID: b.ID, ColumnID: todo.ID, Title: "mid"}
		tt.Position = pos
		require.NoError(t, CreateTask(ctx, conn, tt))
	}

	_, err := RebalanceColumn(ctx, conn, todo.ID)
	require.NoError(t, err)

	rows, _ := conn.QueryContext(ctx, `SELECT position FROM tasks WHERE column_id=? ORDER BY position`, todo.ID)
	var positions []float64
	for rows.Next() {
		var pp float64
		require.NoError(t, rows.Scan(&pp))
		positions = append(positions, pp)
	}
	_ = rows.Close()

	assert.True(t, sort.Float64sAreSorted(positions))
	require.Greater(t, len(positions), 2)
	assert.Equal(t, 1000.0, positions[0])
	assert.Equal(t, PositionStep*float64(len(positions)), positions[len(positions)-1],
		"after rebalance positions should be evenly spaced")
	for i := 1; i < len(positions); i++ {
		gap := positions[i] - positions[i-1]
		assert.GreaterOrEqual(t, gap, PositionStep*0.99,
			"after rebalance gap[%d]=%v should be ~= %v", i, gap, PositionStep)
	}
}

func TestPosition_ConcurrentInsertsDontLoseData(t *testing.T) {
	conn := newTestDB(t)
	ctx := context.Background()
	p, _ := CreateProject(ctx, conn, "p", "")
	boards, _ := ListBoards(ctx, conn, p.ID)
	b := boards[0]
	cols, _ := ListColumns(ctx, conn, b.ID)
	todo := cols[0]

	const workers = 10
	const perWorker = 20

	var wg sync.WaitGroup
	errs := make(chan error, workers*perWorker)

	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for i := 0; i < perWorker; i++ {
				tt := &model.Task{BoardID: b.ID, ColumnID: todo.ID, Title: "task"}
				if err := CreateTask(ctx, conn, tt); err != nil {
					errs <- err
				}
			}
		}(w)
	}
	wg.Wait()
	close(errs)

	for err := range errs {
		t.Errorf("concurrent insert failed: %v", err)
	}

	var total int
	require.NoError(t, conn.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM tasks WHERE column_id=?`, todo.ID).Scan(&total))
	assert.Equal(t, workers*perWorker, total, "no inserts should be lost under concurrency")

	positions := []float64{}
	rows, _ := conn.QueryContext(ctx, `SELECT position FROM tasks WHERE column_id=? ORDER BY position`, todo.ID)
	for rows.Next() {
		var pp float64
		require.NoError(t, rows.Scan(&pp))
		positions = append(positions, pp)
	}
	_ = rows.Close()
	assert.True(t, sort.Float64sAreSorted(positions),
		"all positions must sort cleanly even with concurrent inserts")
}

func TestPosition_GapTooSmallTriggersRebalance(t *testing.T) {
	conn := newTestDB(t)
	ctx := context.Background()
	p, _ := CreateProject(ctx, conn, "p", "")
	boards, _ := ListBoards(ctx, conn, p.ID)
	b := boards[0]
	cols, _ := ListColumns(ctx, conn, b.ID)
	todo := cols[0]

	first := &model.Task{BoardID: b.ID, ColumnID: todo.ID, Title: "a"}
	require.NoError(t, CreateTask(ctx, conn, first))
	second := &model.Task{BoardID: b.ID, ColumnID: todo.ID, Title: "b"}
	require.NoError(t, CreateTask(ctx, conn, second))

	pos := first.Position + 1e-15
	_, err := computeNextPosition(ctx, conn, todo.ID, nil)
	require.NoError(t, err)

	tt := &model.Task{BoardID: b.ID, ColumnID: todo.ID, Title: "tiny"}
	tt.Position = pos
	require.NoError(t, CreateTask(ctx, conn, tt))

	needed, err := needsRebalance(ctx, conn, todo.ID)
	require.NoError(t, err)
	_ = needed

	count, err := RebalanceColumn(ctx, conn, todo.ID)
	require.NoError(t, err)
	assert.Equal(t, 3, count)

	rows, _ := conn.QueryContext(ctx, `SELECT position FROM tasks WHERE column_id=? ORDER BY position`, todo.ID)
	var positions []float64
	for rows.Next() {
		var pp float64
		require.NoError(t, rows.Scan(&pp))
		positions = append(positions, pp)
	}
	_ = rows.Close()
	assert.Equal(t, []float64{1000, 2000, 3000}, positions)
}