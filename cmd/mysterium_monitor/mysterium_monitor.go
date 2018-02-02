package main

import (
	"fmt"
	"github.com/mysterium/node/cmd/commands/monitor"
	"os"
)

func main() {
	cmd := monitor.NewCommand()

	options, err := monitor.ParseArguments(os.Args)
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
