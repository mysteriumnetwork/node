package main

import (
	"fmt"
	"github.com/mysterium/node/cmd/mysterium_server/command_run"
	"os"
)

func main() {
	if err := command_run.NewCommand().Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
