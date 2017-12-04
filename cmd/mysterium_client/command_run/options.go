package command_run

import (
	"errors"
	"flag"
)

type CommandOptions struct {
	NodeKey           string
	DirectoryRuntime  string
	TequilaApiAddress string
	TequilaApiPort    int
}

func ParseArguments(args []string) (options CommandOptions, err error) {
	flags := flag.NewFlagSet(args[0], flag.ContinueOnError)
	flags.StringVar(
		&options.NodeKey,
		"node",
		"",
		"Mysterium VPN node to make connection with",
	)
	flags.StringVar(
		&options.DirectoryRuntime,
		"runtime-dir",
		".",
		"Runtime directory for temp files (should be writable)",
	)
	flags.StringVar(
		&options.TequilaApiAddress,
		"tequilapi.address",
		"",
		"IP address of interface to listen for incoming connections. By default - listen on all interfaces",
	)

	flags.IntVar(
		&options.TequilaApiPort,
		"tequilapi.port",
		4050,
		"Port for listening incoming api requests. By default - 4050",
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
