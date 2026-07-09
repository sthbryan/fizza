package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

const PositionStep = 1000.0

const (
	MinInsertGap    = 1e-9
	RebalanceLimit  = 50
	RebalanceGapRel = 1e-6
	RebalanceWindow = 20
)

func computePositionBefore(ctx context.Context, q Querier, columnID int64, beforeTaskID int64) (float64, error) {
	var beforePos float64
	err := q.QueryRowContext(ctx,
		`SELECT position FROM tasks WHERE id = ? AND column_id = ?`, beforeTaskID, columnID,
	).Scan(&beforePos)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, fmt.Errorf("%w: task %d", ErrNotFound, beforeTaskID)
		}
		return 0, fmt.Errorf("db: get before task: %w", err)
	}
	var prevPos *float64
	err = q.QueryRowContext(ctx,
		`SELECT MAX(position) FROM tasks
		 WHERE column_id = ? AND position < ? AND id != ?`,
		columnID, beforePos, beforeTaskID,
	).Scan(&prevPos)
	if err != nil {
		return 0, fmt.Errorf("db: get prev: %w", err)
	}
	if prevPos == nil {
		return beforePos - PositionStep, nil
	}
	gap := beforePos - *prevPos
	if gap < MinInsertGap {
		return 0, errGapTooSmall
	}
	return (*prevPos + beforePos) / 2, nil
}

func computeNextPosition(ctx context.Context, q Querier, columnID int64, afterTaskID *int64) (float64, error) {
	if afterTaskID != nil {
		var prevPos, nextPos *float64
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
		if prevPos == nil {
			return 0, errors.New("db: afterTaskID not found")
		}
		if nextPos == nil {
			return *prevPos + PositionStep, nil
		}
		gap := *nextPos - *prevPos
		if gap < MinInsertGap {
			return 0, errGapTooSmall
		}
		return (*prevPos + *nextPos) / 2, nil
	}

	var maxPos *float64
	if err := q.QueryRowContext(ctx,
		`SELECT MAX(position) FROM tasks WHERE column_id = ?`, columnID,
	).Scan(&maxPos); err != nil {
		return 0, fmt.Errorf("db: max position: %w", err)
	}
	if maxPos == nil {
		return PositionStep, nil
	}
	return *maxPos + PositionStep, nil
}

var errGapTooSmall = errors.New("db: position gap too small, rebalance required")

func needsRebalance(ctx context.Context, q Querier, columnID int64) (bool, error) {
	var count int
	if err := q.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM tasks WHERE column_id = ?`, columnID,
	).Scan(&count); err != nil {
		return false, err
	}
	if count < RebalanceLimit {
		return false, nil
	}

	var minGap *float64
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
	if minGap == nil {
		return false, nil
	}
	return *minGap < PositionStep*RebalanceGapRel, nil
}

func RebalanceColumn(ctx context.Context, q Querier, columnID int64) (int, error) {
	return rebalanceColumn(ctx, q, columnID, 0)
}

func RebalanceColumnWindow(ctx context.Context, q Querier, columnID, aroundID int64) (int, error) {
	return rebalanceColumn(ctx, q, columnID, aroundID)
}

func rebalanceColumn(ctx context.Context, q Querier, columnID, aroundID int64) (int, error) {
	txer, ok := q.(Transactor)
	if !ok {
		return 0, errors.New("db: rebalance requires *sql.DB or *sql.Tx")
	}
	tx, err := txer.BeginTxx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("db: begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	ids, err := selectRebalanceIDs(ctx, tx, columnID, aroundID)
	if err != nil {
		return 0, err
	}
	if len(ids) == 0 {
		return 0, tx.Commit()
	}

	windowed := aroundID > 0
	var base float64
	if windowed {
		if err := q.QueryRowContext(ctx,
			`SELECT position FROM tasks WHERE id = ?`, aroundID,
		).Scan(&base); err != nil {
			return 0, fmt.Errorf("db: read around position: %w", err)
		}
	}

	stmt, err := tx.PrepareContext(ctx, `UPDATE tasks SET position = ? WHERE id = ?`)
	if err != nil {
		return 0, fmt.Errorf("db: prepare rebalance: %w", err)
	}
	defer stmt.Close()

	for i, id := range ids {
		var newPos float64
		if windowed {
			half := float64(len(ids)/2) * PositionStep
			newPos = base - half + float64(i+1)*PositionStep
		} else {
			newPos = float64(i+1) * PositionStep
		}
		if _, err := stmt.ExecContext(ctx, newPos, id); err != nil {
			return 0, fmt.Errorf("db: rebalance id=%d: %w", id, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("db: commit rebalance: %w", err)
	}
	return len(ids), nil
}

func selectRebalanceIDs(ctx context.Context, q Querier, columnID, aroundID int64) ([]int64, error) {
	if aroundID > 0 {
		half := RebalanceWindow / 2
		rows, err := q.QueryContext(ctx, `
			SELECT id FROM (
				SELECT id, position FROM (
					SELECT id, position FROM tasks
					WHERE column_id = ? AND position >= (SELECT position FROM tasks WHERE id = ?)
					ORDER BY position, id LIMIT ?
				)
				UNION
				SELECT id, position FROM (
					SELECT id, position FROM tasks
					WHERE column_id = ? AND position < (SELECT position FROM tasks WHERE id = ?)
					ORDER BY position DESC, id DESC LIMIT ?
				)
				ORDER BY position, id
			)`,
			columnID, aroundID, half,
			columnID, aroundID, half,
		)
		if err != nil {
			return nil, fmt.Errorf("db: select window: %w", err)
		}
		return scanIDs(rows)
	}
	rows, err := q.QueryContext(ctx,
		`SELECT id FROM tasks WHERE column_id = ? ORDER BY position, id`, columnID)
	if err != nil {
		return nil, fmt.Errorf("db: select all: %w", err)
	}
	return scanIDs(rows)
}

func scanIDs(rows *sql.Rows) ([]int64, error) {
	defer rows.Close()
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func ensureHealthyPosition(ctx context.Context, q Querier, columnID int64) {
	needed, err := needsRebalance(ctx, q, columnID)
	if err != nil || !needed {
		return
	}
	_, _ = RebalanceColumn(ctx, q, columnID)
}
