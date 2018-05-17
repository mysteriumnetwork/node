package main

import (
	"fmt"
	"github.com/mysterium/node/cmd"
	"github.com/mysterium/node/cmd/commands/server"
	_ "github.com/mysterium/node/logconfig"
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

	cmd.StopOnInterruptsConditional(cmd.NewApplicationStopper(serverCommand.Kill), serverCommand.WaitUnregister)

	if err = serverCommand.Wait(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
