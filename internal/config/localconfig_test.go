package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
}

func TestParseLocalConfig_Basic(t *testing.T) {
	f := filepath.Join(t.TempDir(), ".fizza")
	writeFile(t, f, "PROJECT=fizza\nMODE=human\n")
	cfg, err := ParseLocalConfigFile(f)
	require.NoError(t, err)
	assert.Equal(t, "fizza", cfg.Project)
	assert.Equal(t, ModeHuman, cfg.Mode)
}

func TestParseLocalConfig_CommentsAndBlanks(t *testing.T) {
	f := filepath.Join(t.TempDir(), ".fizza")
	writeFile(t, f, "\n# comment\n\nPROJECT=alpha\n  # another\nMODE=llm\n")
	cfg, err := ParseLocalConfigFile(f)
	require.NoError(t, err)
	assert.Equal(t, "alpha", cfg.Project)
	assert.Equal(t, ModeLLM, cfg.Mode)
}

func TestParseLocalConfig_QuotedValues(t *testing.T) {
	f := filepath.Join(t.TempDir(), ".fizza")
	writeFile(t, f, "PROJECT=\"my project\"\nMODE='human'\n")
	cfg, err := ParseLocalConfigFile(f)
	require.NoError(t, err)
	assert.Equal(t, "my project", cfg.Project)
	assert.Equal(t, ModeHuman, cfg.Mode)
}

func TestParseLocalConfig_CaseInsensitiveKey(t *testing.T) {
	f := filepath.Join(t.TempDir(), ".fizza")
	writeFile(t, f, "project=alpha\nmode=Human\n")
	cfg, err := ParseLocalConfigFile(f)
	require.NoError(t, err)
	assert.Equal(t, "alpha", cfg.Project)
	assert.Equal(t, ModeHuman, cfg.Mode)
}

func TestParseLocalConfig_UnknownKey(t *testing.T) {
	f := filepath.Join(t.TempDir(), ".fizza")
	writeFile(t, f, "WAT=foo\n")
	_, err := ParseLocalConfigFile(f)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown key")
}

func TestParseLocalConfig_MissingEquals(t *testing.T) {
	f := filepath.Join(t.TempDir(), ".fizza")
	writeFile(t, f, "PROJECT_ALONE\n")
	_, err := ParseLocalConfigFile(f)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing '='")
}

func TestParseLocalConfig_InvalidMode(t *testing.T) {
	f := filepath.Join(t.TempDir(), ".fizza")
	writeFile(t, f, "MODE=weird\n")
	_, err := ParseLocalConfigFile(f)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid MODE")
}

func TestFindLocalConfig_CwdOnly(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, ".fizza")
	writeFile(t, f, "PROJECT=alpha\n")
	got, ok := FindLocalConfig(dir)
	require.True(t, ok)
	assert.Equal(t, f, got)
}

func TestFindLocalConfig_WalksUpToGit(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(root, ".git"), 0o755))
	f := filepath.Join(root, ".fizza")
	writeFile(t, f, "PROJECT=alpha\n")
	sub := filepath.Join(root, "a", "b", "c")
	require.NoError(t, os.MkdirAll(sub, 0o755))
	got, ok := FindLocalConfig(sub)
	require.True(t, ok)
	assert.Equal(t, f, got)
}

func TestFindLocalConfig_StopsAtGit(t *testing.T) {
	root := t.TempDir()
	other := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(root, ".git"), 0o755))
	f := filepath.Join(other, ".fizza")
	writeFile(t, f, "PROJECT=alpha\n")
	got, ok := FindLocalConfig(root)
	assert.False(t, ok)
	assert.Empty(t, got)
}

func TestFindLocalConfig_DepthCap(t *testing.T) {
	chain := t.TempDir()
	cur := chain
	for i := 0; i < 10; i++ {
		next := filepath.Join(cur, "d")
		require.NoError(t, os.Mkdir(next, 0o755))
		cur = next
	}
	f := filepath.Join(chain, ".fizza")
	writeFile(t, f, "PROJECT=alpha\n")
	got, ok := FindLocalConfig(cur)
	assert.False(t, ok, "should not find .fizza deeper than 5 levels: got %q", got)
}

func TestFindLocalConfig_StopsAtHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	require.NoError(t, os.MkdirAll(filepath.Join(home, "sub"), 0o755))
	f := filepath.Join(home, ".fizza")
	writeFile(t, f, "PROJECT=alpha\n")
	got, ok := FindLocalConfig(filepath.Join(home, "sub"))
	assert.True(t, ok, "should find .fizza in home, got %q", got)
}

func TestFindLocalConfig_PicksClosest(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, ".fizza"), "PROJECT=outer\n")
	sub := filepath.Join(root, "child")
	require.NoError(t, os.Mkdir(sub, 0o755))
	writeFile(t, filepath.Join(sub, ".fizza"), "PROJECT=inner\n")
	got, ok := FindLocalConfig(sub)
	require.True(t, ok)
	assert.True(t, strings.HasSuffix(got, filepath.Join("child", ".fizza")))
}

func TestLoadLocalConfig_MergesWithGlobal(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("HOME", t.TempDir())
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, ".git"), 0o755))
	require.NoError(t, configSaveForTest(filepath.Join(root, ".fizza"), "PROJECT=local-p\n"))

	global := DefaultConfig()
	global.Project = "global-p"
	global.Mode = ModeHuman
	require.NoError(t, SaveConfig(global))

	merged, err := LoadLocalConfig(root)
	require.NoError(t, err)
	assert.Equal(t, "local-p", merged.Project, "local project wins")
}

func configSaveForTest(path, content string) error {
	return os.WriteFile(path, []byte(content), 0o644)
}

func TestLoadEffectiveConfig_PartialOverride(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("HOME", t.TempDir())
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, ".git"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, ".fizza"), []byte("PROJECT=local-p\n"), 0o644))

	global := DefaultConfig()
	global.Project = "global-p"
	global.Mode = ModeHuman
	require.NoError(t, SaveConfig(global))

	eff, err := LoadEffectiveConfig(root)
	require.NoError(t, err)
	assert.Equal(t, "local-p", eff.Project)
	assert.Equal(t, ModeHuman, eff.Mode, "global mode preserved when local unset")
}

func TestLoadEffectiveConfig_NoLocal(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("HOME", t.TempDir())
	root := t.TempDir()

	global := DefaultConfig()
	global.Project = "global-p"
	global.Mode = ModeLLM
	require.NoError(t, SaveConfig(global))

	eff, err := LoadEffectiveConfig(root)
	require.NoError(t, err)
	assert.Equal(t, "global-p", eff.Project)
	assert.Equal(t, ModeLLM, eff.Mode)
}

func TestMergeConfig(t *testing.T) {
	g := Config{Mode: ModeLLM, Project: "g"}
	l := Config{Mode: ModeHuman, Project: "l"}
	assert.Equal(t, Config{Mode: ModeHuman, Project: "l"}, mergeConfig(g, l))
	g2 := Config{Mode: ModeLLM, Project: "g"}
	l2 := Config{Mode: "", Project: "l"}
	assert.Equal(t, Config{Mode: ModeLLM, Project: "l"}, mergeConfig(g2, l2))
}
