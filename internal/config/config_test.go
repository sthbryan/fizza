package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDBPath_FlagWins(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "custom.db")
	got, err := DBPath(target)
	require.NoError(t, err)
	assert.Equal(t, target, got, "flag value must be returned as-is (no rewriting)")
}

func TestDBPath_EnvWhenNoFlag(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "fromenv.db")
	t.Setenv(EnvDB, target)

	got, err := DBPath("")
	require.NoError(t, err)
	assert.Equal(t, target, got)
}

func TestDBPath_DefaultWhenNothingSet(t *testing.T) {
	t.Setenv(EnvDB, "")
	t.Setenv("XDG_CONFIG_HOME", "")

	got, err := DBPath("")
	require.NoError(t, err)

	home, _ := os.UserHomeDir()
	want := filepath.Join(home, ".config", DefaultDirName, DefaultName+".db")
	assert.Equal(t, want, got)
}

func TestDBPath_XDGOverride(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	got, err := DBPath("")
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(dir, DefaultDirName, DefaultName+".db"), got)
}

func TestDBPath_CreatesParentDir(t *testing.T) {
	dir := t.TempDir()
	nested := filepath.Join(dir, "deep", "nested", "fizza.db")
	_, err := DBPath(nested)
	require.NoError(t, err)
	_, err = os.Stat(filepath.Dir(nested))
	require.NoError(t, err, "parent dir must exist after DBPath")
}

func TestDBName(t *testing.T) {
	assert.Equal(t, "alpha", DBName("/some/path/alpha.db"))
	assert.Equal(t, "beta", DBName("/some/path/beta"))
}