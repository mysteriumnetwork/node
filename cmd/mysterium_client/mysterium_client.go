package main

import (
	"fmt"
	"github.com/mysterium/node/cmd/mysterium_client/run"
	"os"
)

func main() {

	options, err := run.ParseArguments(os.Args)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	cmd := run.NewCommand(options)

	err = cmd.Run()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if err = cmd.Wait(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
