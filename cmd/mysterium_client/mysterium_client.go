package main

import (
	"fmt"
	"github.com/MysteriumNetwork/node/cmd/mysterium_client/command_run"
	"os"
)

func main() {
	cmd := command_run.NewCommand()

	options, err := command_run.ParseArguments(os.Args)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if err := cmd.Run(options); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if err = cmd.Wait(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
