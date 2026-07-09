package cli

import (
	"fmt"
	"net"
	"os/exec"
	"runtime"
	"strings"
)

func browserURL(listen string) string {
	host, port, err := net.SplitHostPort(listen)
	if err != nil {
		if strings.HasPrefix(listen, "http://") || strings.HasPrefix(listen, "https://") {
			return listen
		}
		return "http://" + listen
	}
	switch host {
	case "", "0.0.0.0", "::", "[::]":
		host = "127.0.0.1"
	}
	return "http://" + net.JoinHostPort(host, port)
}

func openBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("open browser: %w", err)
	}
	return nil
}
