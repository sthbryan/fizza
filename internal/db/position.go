package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
)

const PositionStep = 1000.0

const (
	MinInsertGap    = 1e-9
	RebalanceLimit  = 50
	RebalanceGapRel = 1e-6
)

func computeNextPosition(ctx context.Context, q querier, columnID int64, afterTaskID *int64) (float64, error) {
	if afterTaskID != nil {
		var prevPos, nextPos sql.NullFloat64
		err := q.QueryRowContext(ctx, `
			SELECT
				(SELECT position FROM tasks WHERE id = ?) AS prev,
				(SELECT MIN(position) FROM tasks
				 WHERE column_id = ? AND position > (SELECT position FROM tasks WHERE id = ?)
				) AS next`,
			*afterTaskID, columnID, *afterTaskID,
		).Scan(&prevPos, &nextPos)
		if err != nil {
			return 0, fmt.Errorf("db: neighbors: %w", err)
		}
		if !prevPos.Valid {
			return 0, errors.New("db: afterTaskID not found")
		}
		if !nextPos.Valid {
			return prevPos.Float64 + PositionStep, nil
		}
		gap := nextPos.Float64 - prevPos.Float64
		if gap < MinInsertGap {
			return 0, errGapTooSmall
		}
		return (prevPos.Float64 + nextPos.Float64) / 2, nil
	}

	var maxPos sql.NullFloat64
	if err := q.QueryRowContext(ctx,
		`SELECT MAX(position) FROM tasks WHERE column_id = ?`, columnID,
	).Scan(&maxPos); err != nil {
		return 0, fmt.Errorf("db: max position: %w", err)
	}
	return maxPos.Float64 + PositionStep, nil
}

var errGapTooSmall = errors.New("db: position gap too small, rebalance required")

func needsRebalance(ctx context.Context, q querier, columnID int64) (bool, error) {
	var count int
	if err := q.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM tasks WHERE column_id = ?`, columnID,
	).Scan(&count); err != nil {
		return false, err
	}
	if count < RebalanceLimit {
		return false, nil
	}

	var minGap sql.NullFloat64
	err := q.QueryRowContext(ctx, `
		SELECT MIN(p2 - p1) FROM (
			SELECT position AS p1,
			       LEAD(position) OVER (ORDER BY position) AS p2
			FROM tasks WHERE column_id = ?
		) WHERE p2 IS NOT NULL`, columnID,
	).Scan(&minGap)
	if err != nil {
		return false, err
	}
	if !minGap.Valid {
		return false, nil
	}
	return minGap.Float64 < PositionStep*RebalanceGapRel, nil
}

func RebalanceColumn(ctx context.Context, q querier, columnID int64) (int, error) {
	txer, ok := q.(transactor)
	if !ok {
		return 0, errors.New("db: RebalanceColumn requires *sql.DB or *sql.Tx")
	}
	tx, err := txer.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("db: begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	rows, err := tx.QueryContext(ctx,
		`SELECT id FROM tasks WHERE column_id = ? ORDER BY position, id`, columnID)
	if err != nil {
		return 0, fmt.Errorf("db: select for rebalance: %w", err)
	}
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			_ = rows.Close()
			return 0, err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return 0, err
	}
	_ = rows.Close()

	if len(ids) == 0 {
		return 0, tx.Commit()
	}

	stmt, err := tx.PrepareContext(ctx, `UPDATE tasks SET position = ? WHERE id = ?`)
	if err != nil {
		return 0, fmt.Errorf("db: prepare rebalance: %w", err)
	}
	defer stmt.Close()

	for i, id := range ids {
		newPos := float64(i+1) * PositionStep
		if _, err := stmt.ExecContext(ctx, newPos, id); err != nil {
			return 0, fmt.Errorf("db: rebalance id=%d: %w", id, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("db: commit rebalance: %w", err)
	}
	return len(ids), nil
}

func ensureHealthyPosition(ctx context.Context, q querier, columnID int64) {
	needed, err := needsRebalance(ctx, q, columnID)
	if err != nil || !needed {
		return
	}
	_, _ = RebalanceColumn(ctx, q, columnID)
}

func positionsEqual(a, b float64) bool {
	return math.Abs(a-b) < 1e-12
}