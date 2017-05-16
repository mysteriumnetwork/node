package command_run

import (
	"flag"
)

type CommandOptions struct {
	DirectoryRuntime string
}

func ParseArguments(args []string) (options CommandOptions, err error) {
	flags := flag.NewFlagSet(args[0], flag.ContinueOnError)
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

	return options, err
}
