package config

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const (
	LocalConfigName  = ".fizza"
	maxSearchDepth   = 5
)

func LoadLocalConfig(startDir string) (Config, error) {
	cfg := DefaultConfig()
	path, found := FindLocalConfig(startDir)
	if !found {
		return cfg, nil
	}
	parsed, err := ParseLocalConfigFile(path)
	if err != nil {
		return cfg, err
	}
	if parsed.Project != "" {
		cfg.Project = parsed.Project
	}
	if parsed.Mode != "" {
		cfg.Mode = parsed.Mode
	}
	return cfg, nil
}

func FindLocalConfig(startDir string) (string, bool) {
	dir, err := filepath.Abs(startDir)
	if err != nil {
		return "", false
	}
	home, _ := os.UserHomeDir()
	for i := 0; i < maxSearchDepth; i++ {
		candidate := filepath.Join(dir, LocalConfigName)
		if _, err := os.Stat(candidate); err == nil {
			return candidate, true
		}
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return "", false
		}
		if home != "" && dir == home {
			return "", false
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", false
		}
		dir = parent
	}
	return "", false
}

type localParseError struct {
	path string
	line int
	msg  string
}

func (e *localParseError) Error() string {
	return fmt.Sprintf("config: %s:%d: %s", e.path, e.line, e.msg)
}

func ParseLocalConfigFile(path string) (Config, error) {
	cfg := Config{}
	f, err := os.Open(path)
	if err != nil {
		return cfg, fmt.Errorf("config: open %q: %w", path, err)
	}
	defer f.Close()
	if err := parseLocalConfig(f, path, &cfg); err != nil {
		return cfg, err
	}
	return cfg, nil
}

func parseLocalConfig(r io.Reader, source string, cfg *Config) error {
	scanner := bufio.NewScanner(r)
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		raw := strings.TrimSpace(scanner.Text())
		if raw == "" || strings.HasPrefix(raw, "#") {
			continue
		}
		eq := strings.IndexByte(raw, '=')
		if eq < 0 {
			return &localParseError{path: source, line: lineNo, msg: fmt.Sprintf("missing '=' in %q", raw)}
		}
		key := strings.ToUpper(strings.TrimSpace(raw[:eq]))
		val := strings.TrimSpace(raw[eq+1:])
		val = strings.Trim(val, `"'`)
		switch key {
		case "PROJECT":
			cfg.Project = val
		case "MODE":
			cfg.Mode = strings.ToLower(val)
		default:
			return &localParseError{path: source, line: lineNo, msg: fmt.Sprintf("unknown key %q (want PROJECT or MODE)", key)}
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("config: read %q: %w", source, err)
	}
	if cfg.Mode != "" && cfg.Mode != ModeLLM && cfg.Mode != ModeHuman {
		return &localParseError{path: source, line: 0, msg: fmt.Sprintf("invalid MODE %q (want llm or human)", cfg.Mode)}
	}
	return nil
}

var ErrLocalConfigInvalid = errors.New("local config invalid")
