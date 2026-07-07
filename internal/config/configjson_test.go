package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func withTempHome(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("HOME", dir)
}

func TestConfig_LoadMissing(t *testing.T) {
	withTempHome(t)
	cfg, err := LoadConfig()
	require.NoError(t, err)
	assert.Equal(t, ModeLLM, cfg.Mode)
	assert.Equal(t, "", cfg.Project)
}

func TestConfig_SaveLoadRoundtrip(t *testing.T) {
	withTempHome(t)
	in := Config{Mode: ModeHuman, Project: "alpha"}
	require.NoError(t, SaveConfig(in))
	out, err := LoadConfig()
	require.NoError(t, err)
	assert.Equal(t, in, out)
}

func TestConfig_InvalidJSON(t *testing.T) {
	withTempHome(t)
	path, err := ConfigPath()
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte("not json"), 0o644))
	_, err = LoadConfig()
	assert.Error(t, err)
}

func TestConfig_InvalidMode(t *testing.T) {
	withTempHome(t)
	path, err := ConfigPath()
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	b, _ := json.Marshal(Config{Mode: "wat", Project: "x"})
	require.NoError(t, os.WriteFile(path, b, 0o644))
	_, err = LoadConfig()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid mode")
}

func TestResolveMode(t *testing.T) {
	cases := []struct {
		name       string
		flagFormat string
		cfgMode    string
		wantFormat string
	}{
		{"llm default", "", ModeLLM, "toon"},
		{"human default", "", ModeHuman, "pretty"},
		{"flag format wins", "pretty", ModeLLM, "pretty"},
		{"flag toon wins", "toon", ModeHuman, "toon"},
		{"explicit json", "json", ModeLLM, "json"},
		{"empty config mode", "", "", "json"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ResolveMode(tc.flagFormat, tc.cfgMode)
			assert.Equal(t, tc.wantFormat, got)
		})
	}
}
