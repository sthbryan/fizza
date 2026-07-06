package db

import (
	"context"
	"database/sql"

	"github.com/jmoiron/sqlx"
)

type Querier interface {
	sqlx.ExtContext
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

type Transactor interface {
	Querier
	BeginTxx(ctx context.Context, opts *sql.TxOptions) (*sqlx.Tx, error)
}

func WrapDB(conn *sql.DB) Querier { return sqlx.NewDb(conn, "sqlite") }

func AsTransactor(q Querier) (Transactor, bool) {
	t, ok := q.(Transactor)
	return t, ok
}