package db

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpen_InMemory(t *testing.T) {
	conn, err := Open(context.Background(), ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })

	versions, err := AppliedVersions(context.Background(), conn)
	require.NoError(t, err)
	assert.Equal(t, []int64{1, 2, 3, 4, 5}, versions)

	for _, table := range []string{"projects", "boards", "columns", "tasks", "events", "tags", "task_tags", "schema_migrations"} {
		var name string
		err := conn.QueryRow(
			`SELECT name FROM sqlite_master WHERE type='table' AND name=?`, table,
		).Scan(&name)
		require.NoError(t, err, "table %s should exist", table)
		assert.Equal(t, table, name)
	}
}

func TestOpen_Idempotent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "fizza.db")

	conn1, err := Open(context.Background(), path)
	require.NoError(t, err)
	_, err = conn1.Exec(`INSERT INTO projects (name) VALUES (?)`, "smoke")
	require.NoError(t, err)
	_ = conn1.Close()

	conn2, err := Open(context.Background(), path)
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn2.Close() })

	var n int
	require.NoError(t, conn2.QueryRow(`SELECT COUNT(*) FROM projects`).Scan(&n))
	assert.Equal(t, 1, n)

	versions, err := AppliedVersions(context.Background(), conn2)
	require.NoError(t, err)
	assert.Equal(t, []int64{1, 2, 3, 4, 5}, versions)
}

func TestOpen_EmptyPathFails(t *testing.T) {
	_, err := Open(context.Background(), "")
	require.Error(t, err)
}

func TestLoadMigrations_SortedAndNumeric(t *testing.T) {
	migs, err := LoadMigrations()
	require.NoError(t, err)
	require.NotEmpty(t, migs)
	for i := 1; i < len(migs); i++ {
		assert.Less(t, migs[i-1].Version, migs[i].Version, "migrations must be sorted")
	}
}

func TestOpen_FinalSchemaShape(t *testing.T) {
	conn, err := Open(context.Background(), ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })

	row := conn.QueryRow(`SELECT type FROM pragma_table_info('tasks') WHERE name='position'`)
	var colType string
	require.NoError(t, row.Scan(&colType))
	assert.Equal(t, "REAL", colType)

	row = conn.QueryRow(`
		SELECT on_delete FROM pragma_foreign_key_list('tasks')
		WHERE "table" = 'boards' LIMIT 1`)
	var rule string
	require.NoError(t, row.Scan(&rule))
	assert.Equal(t, "RESTRICT", rule)

	row = conn.QueryRow(`
		SELECT on_delete FROM pragma_foreign_key_list('tasks')
		WHERE "table" = 'columns' LIMIT 1`)
	require.NoError(t, row.Scan(&rule))
	assert.Equal(t, "RESTRICT", rule)
}

func TestMigrationStatus(t *testing.T) {
	conn, err := Open(context.Background(), ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })
	pending, applied, err := MigrationStatus(context.Background(), conn)
	require.NoError(t, err)
	assert.Empty(t, pending)
	assert.NotEmpty(t, applied)
	for _, a := range applied {
		assert.NotEmpty(t, a.Checksum, "applied migrations must have a stored checksum")
		assert.NotEmpty(t, a.AppliedAt)
	}
}
