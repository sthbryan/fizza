package mcp

import "os/exec"

func commandForServer(xdgConfig string) *exec.Cmd {
	cmd := exec.Command(testBinPath, "mcp")
	cmd.Env = append(cmd.Environ(), "XDG_CONFIG_HOME="+xdgConfig)
	return cmd
}
