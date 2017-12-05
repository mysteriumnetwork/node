package command_run

import (
	"flag"
	"github.com/mysterium/node/utils/file"
)

type CommandOptions struct {
	NodeKey           string
	DirectoryConfig   string
	DirectoryRuntime  string
	DirectoryKeystore string
}

func ParseArguments(args []string) (options CommandOptions, err error) {
	flags := flag.NewFlagSet(args[0], flag.ContinueOnError)
	flags.StringVar(
		&options.NodeKey,
		"node",
		"",
		"Unique identifier for Mysterium VPN node",
	)

	flags.StringVar(
		&options.DirectoryConfig,
		"config-dir",
		file.GetMysteriumDirectory("config"),
		"Configs directory containing all configuration files",
	)

	flags.StringVar(
		&options.DirectoryRuntime,
		"runtime-dir",
		file.GetMysteriumDirectory("run"),
		"Runtime directory for temp files (should be writable)",
	)

	flags.StringVar(
		&options.DirectoryKeystore,
		"keystore-dir",
		file.GetMysteriumDirectory("keystore"),
		"Keystore directory",
	)

	err = flags.Parse(args[1:])
	if err != nil {
		return
	}

	return options, err
}
