package command_run

import (
	"errors"
	"flag"
)

type commandRunOptions struct {
	NodeKey         string
	DirectoryConfig string
}

func (cmd *commandRun) parseArguments(args []string) (options commandRunOptions, err error) {
	flags := flag.NewFlagSet(args[0], flag.ContinueOnError)
	flags.StringVar(
		&options.NodeKey,
		"node",
		"12345",
		"Unique identifier for my Mysterium VPN node",
	)
	flags.StringVar(
		&options.DirectoryConfig,
		"config-dir",
		".",
		"Configs directory containing all configuration files",
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
