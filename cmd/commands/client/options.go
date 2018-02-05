package client

import (
	"flag"
	"github.com/mysterium/node/utils/file"
)

// CommandOptions describes options which are required to start Command
type CommandOptions struct {
	DirectoryConfig   string
	DirectoryRuntime  string
	DirectoryKeystore string

	TequilapiAddress string
	TequilapiPort    int

	CLI bool
}

// ParseArguments parses CLI flags and adds to CommandOptions structure
func ParseArguments(args []string) (options CommandOptions, err error) {
	flags := flag.NewFlagSet(args[0], flag.ContinueOnError)
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
		"Runtime writable directory for temp files",
	)
	flags.StringVar(
		&options.DirectoryKeystore,
		"keystore-dir",
		file.GetMysteriumDirectory("keystore"),
		"Keystore directory",
	)

	flags.StringVar(
		&options.TequilapiAddress,
		"tequilapi.address",
		"localhost",
		"IP address of interface to listen for incoming connections",
	)
	flags.IntVar(
		&options.TequilapiPort,
		"tequilapi.port",
		4050,
		"Port for listening incoming api requests",
	)

	flags.BoolVar(
		&options.CLI,
		"cli",
		false,
		"Run an interactive CLI based Mysterium UI",
	)

	err = flags.Parse(args[1:])
	if err != nil {
		return
	}

	return options, err
}
