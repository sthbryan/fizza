package main

import (
	"fmt"
	"os"

	"github.com/fizza/fizza/internal/cli"
)

var version = "dev"

func main() {
	cli.SetVersion(version)
	code, err := cli.ExecuteWithCode()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
	os.Exit(code)
}