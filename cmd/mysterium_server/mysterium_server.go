package main

import (
	"os"
	"fmt"
	"github.com/mysterium/node/cmd/mysterium_server/command_run"
)

func main() {
	command := command_run.NewCommandRun()
	if err := command.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
