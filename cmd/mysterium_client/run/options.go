package run

import (
	"flag"
	"github.com/mysterium/node/utils/file"
)

type CommandOptions struct {
	DirectoryRuntime  string
	DirectoryKeystore string
	TequilapiAddress  string
	TequilapiPort     int
	InteractiveCli    bool
}

func ParseArguments(args []string) (options CommandOptions, err error) {
	flags := flag.NewFlagSet(args[0], flag.ContinueOnError)
	flags.StringVar(
		&options.DirectoryRuntime,
		"runtime-dir",
		".",
		"Runtime directory for temp files (should be writable)",
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
		"IP address of interface to listen for incoming connections. By default - bind to local interface",
	)

	flags.IntVar(
		&options.TequilapiPort,
		"tequilapi.port",
		4050,
		"Port for listening incoming api requests. By default - 4050",
	)

	flags.BoolVar(
		&options.InteractiveCli,
		"interactive-cli",
		false,
		"Run an interactive CLI client",
	)

	err = flags.Parse(args[1:])
	if err != nil {
		return
	}

	return options, err
}
