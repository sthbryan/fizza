package db

import (
	"context"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpenPool_TwoHandles(t *testing.T) {
	dir := t.TempDir()
	pool, err := OpenPool(context.Background(), filepath.Join(dir, "fizza.db"), 2, 1)
	require.NoError(t, err)
	t.Cleanup(func() { _ = pool.Close() })

	assert.NotNil(t, pool.Write)
	assert.NotNil(t, pool.Read)
	assert.NotSame(t, pool.Write, pool.Read)

	var v int
	require.NoError(t, pool.Read.QueryRow(`SELECT 1`).Scan(&v))
	assert.Equal(t, 1, v)
}

func TestOpenPool_ConcurrentReaders(t *testing.T) {
	dir := t.TempDir()
	pool, err := OpenPool(context.Background(), filepath.Join(dir, "fizza.db"), 4, 1)
	require.NoError(t, err)
	t.Cleanup(func() { _ = pool.Close() })

	_, err = pool.Write.Exec(`CREATE TABLE t (n INTEGER)`)
	require.NoError(t, err)
	_, err = pool.Write.Exec(`INSERT INTO t VALUES (1)`)
	require.NoError(t, err)

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var v int
			assert.NoError(t, pool.Read.QueryRow(`SELECT n FROM t`).Scan(&v))
		}()
	}
	wg.Wait()
}

func TestOpenPool_BadPath(t *testing.T) {
	_, err := OpenPool(context.Background(), "", 1, 1)
	assert.Error(t, err)
}