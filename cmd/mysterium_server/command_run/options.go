package command_run

import (
	"flag"
	"github.com/mysterium/node/utils/file"
)

// CommandOptions describes options which are required to start CommandRun
type CommandOptions struct {
	NodeKey           string
	DirectoryConfig   string
	DirectoryRuntime  string
	DirectoryKeystore string
	Passphrase        string

	LocationCountry  string
	LocationDatabase string
}

// ParseArguments parses CLI flags and adds to CommandOptions structure
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

	flags.StringVar(
		&options.Passphrase,
		"passphrase",
		"",
		"Identity passphrase",
	)

	flags.StringVar(
		&options.LocationDatabase,
		"location.database",
		"GeoLite2-Country.mmdb",
		"Service location autodetect database (GeoLite2 formatted e.g. http://dev.maxmind.com/geoip/geoip2/geolite2/)",
	)
	flags.StringVar(
		&options.LocationCountry,
		"location.country",
		"",
		"Service location country. If not given country is autodetected",
	)

	err = flags.Parse(args[1:])
	if err != nil {
		return
	}

	return options, err
}
