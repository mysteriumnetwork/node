package command_run

import (
	"errors"
	"flag"
)

type CommandOptions struct {
	DirectoryRuntime string
	NodeKeys         []string
}

func ParseArguments(args []string) (options CommandOptions, err error) {
	flags := flag.NewFlagSet(args[0], flag.ContinueOnError)
	flags.StringVar(
		&options.DirectoryRuntime,
		"runtime-dir",
		".",
		"Runtime directory for temp files (should be writable)",
	)

	var nodeToCheck string
	flags.StringVar(
		&nodeToCheck,
		"node",
		"",
		"Which node should be checked",
	)

	err = flags.Parse(args[1:])
	if err != nil {
		return
	}

	if nodeToCheck != "" {
		options.NodeKeys = []string{nodeToCheck}
	} else {
		err = errors.New("Provide which nodes to monitor!")
	}

	return options, err
}
