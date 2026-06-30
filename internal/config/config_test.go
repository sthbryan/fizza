package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDBPath_DefaultWhenNothingSet(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "")

	got, err := DBPath()
	require.NoError(t, err)

	home, _ := os.UserHomeDir()
	want := filepath.Join(home, ".config", DefaultDirName, DefaultName+".db")
	assert.Equal(t, want, got)
}

func TestDBPath_XDGOverride(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	got, err := DBPath()
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(dir, DefaultDirName, DefaultName+".db"), got)
}

func TestDBPath_CreatesParentDir(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	_, err := DBPath()
	require.NoError(t, err)

	want := filepath.Join(dir, DefaultDirName)
	stat, err := os.Stat(want)
	require.NoError(t, err)
	assert.True(t, stat.IsDir(), "fizza config dir must exist after DBPath")
}

func TestDBName(t *testing.T) {
	assert.Equal(t, "alpha", DBName("/some/path/alpha.db"))
	assert.Equal(t, "beta", DBName("/some/path/beta"))
}
