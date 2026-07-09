package db

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"embed"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"
)

const walDSN = "file:%s?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=synchronous(NORMAL)"

//go:embed migrations/*.sql
var migrationsFS embed.FS

type Migration struct {
	Version int64
	Name    string
	Body    string
}

func Open(ctx context.Context, path string) (*sqlx.DB, error) {
	if strings.TrimSpace(path) == "" {
		return nil, errors.New("db: empty path")
	}
	dsn := fmt.Sprintf(walDSN, filepath.Clean(path))
	raw, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("db: open: %w", err)
	}
	raw.SetMaxOpenConns(1)
	raw.SetMaxIdleConns(1)
	raw.SetConnMaxLifetime(0)

	conn := sqlx.NewDb(raw, "sqlite")
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := conn.PingContext(pingCtx); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("db: ping: %w", err)
	}
	if err := Migrate(ctx, conn); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("db: migrate: %w", err)
	}
	return conn, nil
}

type Pool struct {
	Write *sqlx.DB
	Read  *sqlx.DB
	path  string
}

func OpenPool(ctx context.Context, path string, maxReaders, maxWriters int) (*Pool, error) {
	if strings.TrimSpace(path) == "" {
		return nil, errors.New("db: empty path")
	}
	if maxReaders < 1 {
		maxReaders = 1
	}
	if maxWriters < 1 {
		maxWriters = 1
	}
	dsn := fmt.Sprintf(walDSN, filepath.Clean(path))

	writer, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("db: open writer: %w", err)
	}
	writer.SetMaxOpenConns(maxWriters)
	writer.SetMaxIdleConns(maxWriters)
	writer.SetConnMaxLifetime(0)

	reader, err := sql.Open("sqlite", dsn)
	if err != nil {
		_ = writer.Close()
		return nil, fmt.Errorf("db: open reader: %w", err)
	}
	reader.SetMaxOpenConns(maxReaders)
	reader.SetMaxIdleConns(maxReaders)
	reader.SetConnMaxLifetime(0)

	wx := sqlx.NewDb(writer, "sqlite")
	rx := sqlx.NewDb(reader, "sqlite")

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := writer.PingContext(pingCtx); err != nil {
		_ = writer.Close()
		_ = reader.Close()
		return nil, fmt.Errorf("db: ping writer: %w", err)
	}
	if err := reader.PingContext(pingCtx); err != nil {
		_ = writer.Close()
		_ = reader.Close()
		return nil, fmt.Errorf("db: ping reader: %w", err)
	}
	if err := Migrate(ctx, sqlx.NewDb(writer, "sqlite")); err != nil {
		_ = writer.Close()
		_ = reader.Close()
		return nil, fmt.Errorf("db: migrate: %w", err)
	}
	return &Pool{Write: wx, Read: rx, path: filepath.Clean(path)}, nil
}

func (p *Pool) Close() error {
	if p == nil {
		return nil
	}
	if p.Write == p.Read {
		if p.Write == nil {
			return nil
		}
		return p.Write.Close()
	}
	var werr, rerr error
	if p.Write != nil {
		werr = p.Write.Close()
	}
	if p.Read != nil {
		rerr = p.Read.Close()
	}
	if werr != nil {
		return werr
	}
	return rerr
}

func (p *Pool) Path() string {
	if p == nil {
		return ""
	}
	return p.path
}

func NewSinglePool(conn *sqlx.DB) *Pool {
	return &Pool{Write: conn, Read: conn, path: ""}
}

func LoadMigrations() ([]Migration, error) {
	entries, err := fs.ReadDir(migrationsFS, "migrations")
	if err != nil {
		return nil, fmt.Errorf("read migrations dir: %w", err)
	}
	var out []Migration
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".sql") {
			continue
		}
		idx := strings.IndexByte(e.Name(), '_')
		if idx <= 0 {
			return nil, fmt.Errorf("migration %q: missing numeric prefix", e.Name())
		}
		num, err := strconv.ParseInt(e.Name()[:idx], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("migration %q: bad version: %w", e.Name(), err)
		}
		body, err := fs.ReadFile(migrationsFS, "migrations/"+e.Name())
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", e.Name(), err)
		}
		out = append(out, Migration{Version: num, Name: e.Name(), Body: string(body)})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Version < out[j].Version })
	return out, nil
}

func Migrate(ctx context.Context, conn Querier) error {
	migs, err := LoadMigrations()
	if err != nil {
		return err
	}
	if _, err := conn.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
		version    INTEGER PRIMARY KEY,
		applied_at TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
		checksum   TEXT    NOT NULL DEFAULT ''
	)`); err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}
	applied := map[int64]bool{}
	checksums := map[int64]string{}
	rows, err := conn.QueryContext(ctx, `SELECT version, COALESCE(checksum,'') FROM schema_migrations`)
	if err != nil {
		return fmt.Errorf("select applied: %w", err)
	}
	for rows.Next() {
		var v int64
		var cs string
		if err := rows.Scan(&v, &cs); err != nil {
			_ = rows.Close()
			return fmt.Errorf("scan applied: %w", err)
		}
		applied[v] = true
		checksums[v] = cs
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return fmt.Errorf("rows applied: %w", err)
	}
	_ = rows.Close()

	for _, m := range migs {
		stored, ok := checksums[m.Version]
		if !ok || stored == "" {
			continue
		}
		sum := sha256.Sum256([]byte(m.Body))
		actual := hex.EncodeToString(sum[:])
		if actual != stored {
			return fmt.Errorf("migration %s checksum mismatch (want %s, got %s); the migration file has been modified after being applied",
				m.Name, stored, actual)
		}
	}

	for _, m := range migs {
		if applied[m.Version] {
			continue
		}
		tr, ok := conn.(Transactor)
		if !ok {
			return fmt.Errorf("migration runner requires *sqlx.DB or *sqlx.Tx")
		}
		tx, err := tr.BeginTxx(ctx, nil)
		if err != nil {
			return fmt.Errorf("begin tx for %s: %w", m.Name, err)
		}
		if _, err := tx.ExecContext(ctx, m.Body); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("exec %s: %w", m.Name, err)
		}
		sum := sha256.Sum256([]byte(m.Body))
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO schema_migrations (version, checksum) VALUES (?, ?)`,
			m.Version, hex.EncodeToString(sum[:])); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("record %s: %w", m.Name, err)
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit %s: %w", m.Name, err)
		}
	}
	return nil
}

func AppliedVersions(ctx context.Context, conn Querier) ([]int64, error) {
	rows, err := conn.QueryContext(ctx, `SELECT version FROM schema_migrations ORDER BY version`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []int64
	for rows.Next() {
		var v int64
		if err := rows.Scan(&v); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

type SchemaObject struct {
	Type string `json:"type"`
	Name string `json:"name"`
	SQL  string `json:"sql"`
}

func Schema(ctx context.Context, conn Querier) ([]SchemaObject, error) {
	rows, err := conn.QueryContext(ctx,
		`SELECT type, name, sql FROM sqlite_master WHERE name NOT LIKE 'sqlite_%' ORDER BY type, name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []SchemaObject
	for rows.Next() {
		var s SchemaObject
		var sqlText *string
		if err := rows.Scan(&s.Type, &s.Name, &sqlText); err != nil {
			return nil, err
		}
		if sqlText != nil {
			s.SQL = *sqlText
		}
		out = append(out, s)
	}
	return out, rows.Err()
}
