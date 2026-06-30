package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const DefaultDirName = "fizza"

const DefaultName = "default"

func DBPath() (string, error) {
	dir, err := configDir()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("config: create dir %q: %w", dir, err)
	}
	return filepath.Join(dir, DefaultName+".db"), nil
}

func DBName(path string) string {
	base := filepath.Base(path)
	return strings.TrimSuffix(base, filepath.Ext(base))
}

func configDir() (string, error) {
	if xdg := strings.TrimSpace(os.Getenv("XDG_CONFIG_HOME")); xdg != "" {
		return filepath.Join(xdg, DefaultDirName), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("config: resolve home dir: %w", err)
	}
	return filepath.Join(home, ".config", DefaultDirName), nil
}
