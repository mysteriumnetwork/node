package main

import (
	"fmt"
	"github.com/mysterium/node/cmd/mysterium_client/command_run"
	"os"
)

func main() {
	command := command_run.NewCommandRun()
	if err := command.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
