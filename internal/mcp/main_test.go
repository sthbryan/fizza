package mcp

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

var testBinPath string

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "fizza-mcp-test")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(dir)

	testBinPath = filepath.Join(dir, "fizza")
	build := exec.Command("go", "build", "-o", testBinPath, "../../cmd/fizza")
	build.Stdout = os.Stderr
	build.Stderr = os.Stderr
	if err := build.Run(); err != nil {
		panic("build test binary: " + err.Error())
	}

	os.Exit(m.Run())
}