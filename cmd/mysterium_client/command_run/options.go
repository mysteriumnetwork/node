package command_run

import (
	"flag"
)

type commandRunOptions struct {
	DirectoryRuntime string
}

func (cmd *commandRun) parseArguments(args []string) (options commandRunOptions, err error) {
	flags := flag.NewFlagSet(args[0], flag.ContinueOnError)
	flags.Bool("help", false, "Show options")
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