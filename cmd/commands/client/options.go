package client

import (
	"flag"
	"github.com/mysterium/node/cmd"
	"path/filepath"
)

// CommandOptions describes options which are required to start Command
type CommandOptions struct {
	DirectoryConfig  string
	DirectoryRuntime string
	DirectoryData    string

	TequilapiAddress string
	TequilapiPort    int

	CLI bool

	DiscoveryAPIAddress string
	BrokerAddress       string
}

var mysteriumApiUrl string
var natsServerIP string

// ParseArguments parses CLI flags and adds to CommandOptions structure
func ParseArguments(args []string) (options CommandOptions, err error) {
	flags := flag.NewFlagSet(args[0], flag.ContinueOnError)
	flags.StringVar(
		&options.DirectoryData,
		"data-dir",
		cmd.GetDataDirectory(),
		"Data directory containing keystore & other persistent files",
	)
	flags.StringVar(
		&options.DirectoryConfig,
		"config-dir",
		filepath.Join(cmd.GetDataDirectory(), "config"),
		"Configs directory containing all configuration files",
	)
	flags.StringVar(
		&options.DirectoryRuntime,
		"runtime-dir",
		filepath.Join(cmd.GetDataDirectory(), "run"),
		"Runtime writable directory for temp files",
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

	flags.StringVar(
		&options.DiscoveryAPIAddress,
		"discovery-address",
		mysteriumApiUrl,
		"Address (URL form) of discovery service",
	)
	flags.StringVar(
		&options.BrokerAddress,
		"broker-address",
		natsServerIP,
		"Address (IP or domain name) of message broker",
	)

	err = flags.Parse(args[1:])
	if err != nil {
		return
	}

	return options, err
}
