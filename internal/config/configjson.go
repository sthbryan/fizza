package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const ConfigFileName = "config.json"

const (
	ModeLLM   = "llm"
	ModeHuman = "human"
)

type Config struct {
	Mode    string `json:"mode"`
	Project string `json:"project"`
	Board   string `json:"board,omitempty"`
}

func DefaultConfig() Config {
	return Config{Mode: ModeLLM, Project: ""}
}

func ConfigPath() (string, error) {
	dir, err := configDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, ConfigFileName), nil
}

func LoadConfig() (Config, error) {
	return loadGlobalConfig()
}

func loadGlobalConfig() (Config, error) {
	cfg := DefaultConfig()
	path, err := ConfigPath()
	if err != nil {
		return cfg, err
	}
	b, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return cfg, nil
		}
		return cfg, fmt.Errorf("config: read %q: %w", path, err)
	}
	if len(b) == 0 {
		return cfg, nil
	}
	if err := json.Unmarshal(b, &cfg); err != nil {
		return cfg, fmt.Errorf("config: parse %q: %w", path, err)
	}
	cfg.Mode = strings.ToLower(strings.TrimSpace(cfg.Mode))
	if cfg.Mode == "" {
		cfg.Mode = ModeLLM
	}
	if cfg.Mode != ModeLLM && cfg.Mode != ModeHuman {
		return cfg, fmt.Errorf("config: invalid mode %q (want llm or human)", cfg.Mode)
	}
	return cfg, nil
}

func LoadEffectiveConfig(startDir string) (Config, error) {
	global, err := loadGlobalConfig()
	if err != nil {
		return global, err
	}
	local, err := LoadLocalConfig(startDir)
	if err != nil {
		return global, err
	}
	return mergeConfig(global, local), nil
}

func mergeConfig(global, local Config) Config {
	out := global
	if local.Project != "" {
		out.Project = local.Project
	}
	if local.Board != "" {
		out.Board = local.Board
	}
	if local.Mode != "" {
		out.Mode = local.Mode
	}
	return out
}

func SaveConfig(cfg Config) error {
	path, err := ConfigPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("config: create dir %q: %w", filepath.Dir(path), err)
	}
	b, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return fmt.Errorf("config: write %q: %w", tmp, err)
	}
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("config: rename %q: %w", path, err)
	}
	return nil
}

func ResolveMode(flagFormat string, cfgMode string) string {
	if flagFormat != "" && flagFormat != "json" {
		return flagFormat
	}
	if cfgMode == ModeHuman {
		return "pretty"
	}
	return "json"
}
