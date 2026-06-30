package main

import (
	"os"

	"github.com/geekjourneyx/md2x/internal/cli"
)

func main() {
	err := cli.Execute()
	os.Exit(cli.ExitCode(err))
}
