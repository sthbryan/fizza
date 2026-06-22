package mcp

import "os/exec"

func commandForServer(dbPath string) *exec.Cmd {
	return exec.Command(testBinPath, "mcp", "--db", dbPath)
}