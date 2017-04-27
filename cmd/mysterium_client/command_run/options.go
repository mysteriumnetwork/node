package command_run

import (
	"errors"
	"flag"
)

type CommandOptions struct {
	NodeKey          string
	DirectoryRuntime string
}

func ParseArguments(args []string) (options CommandOptions, err error) {
	flags := flag.NewFlagSet(args[0], flag.ContinueOnError)
	flags.StringVar(
		&options.NodeKey,
		"node",
		"12345",
		"Mysterium VPN node to make connection with",
	)
	flags.StringVar(
		&options.DirectoryRuntime,
		"runtime-dir",
		".",
		"Runtime directory for temp files (should be writable)",
	)

	err = flags.Parse(args[1:])
	if err != nil {
		return
	}

	if options.NodeKey == "" {
		err = errors.New("Missing VPN node key!")
		return
	}

	return options, err
}
