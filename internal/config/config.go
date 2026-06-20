package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const EnvDB = "FIZZA_DB"

const DefaultDirName = "fizza"

const DefaultName = "default"

func DBPath(flagValue string) (string, error) {
	if p := strings.TrimSpace(flagValue); p != "" {
		return ensureDir(filepath.Dir(p))
	}
	if env := strings.TrimSpace(os.Getenv(EnvDB)); env != "" {
		return ensureDir(filepath.Dir(env))
	}
	dir, err := configDir()
	if err != nil {
		return "", err
	}
	return ensureDir(dir)
}

func DBName(path string) string {
	base := filepath.Base(path)
	return strings.TrimSuffix(base, filepath.Ext(base))
}

func ensureDir(dir string) (string, error) {
	if dir == "" || dir == "." {
		return "", errors.New("config: invalid DB directory")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("config: create dir %q: %w", dir, err)
	}
	return dir, nil
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
