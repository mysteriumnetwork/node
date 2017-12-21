package main

import (
	"fmt"
	"github.com/mysterium/node/cmd/mysterium_client/command_run"
	"os"
)

func main() {

	options, err := command_run.ParseArguments(os.Args)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	cmd, err := command_run.NewCommand(options)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	cmd.Run()

	if err = cmd.Wait(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
