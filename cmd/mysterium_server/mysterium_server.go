package main

import (
	"fmt"
	"github.com/mysterium/node/cmd"
	"github.com/mysterium/node/cmd/commands/server"
	"os"
)

func main() {
	options, err := server.ParseArguments(os.Args)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	serverCommand := server.NewCommand(options)

	if err := serverCommand.Start(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	cmd.StopOnInterrupts(cmd.NewApplicationStopper(serverCommand.Kill))

	if err = serverCommand.Wait(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
