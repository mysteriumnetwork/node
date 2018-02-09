package server

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

	Identity   string
	Passphrase string

	LocationCountry  string
	LocationDatabase string
}

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
		&options.Identity,
		"identity",
		"",
		"Keystore's identity used to provide service. If not given identity will be created automatically",
	)
	flags.StringVar(
		&options.Passphrase,
		"identity.passphrase",
		"",
		"Used to unlock keystore's identity",
	)

	flags.StringVar(
		&options.LocationDatabase,
		"location.database",
		"GeoLite2-Country.mmdb",
		"Service location autodetect database of GeoLite2 format e.g. http://dev.maxmind.com/geoip/geoip2/geolite2/",
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
