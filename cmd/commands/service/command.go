/*
 * Copyright (C) 2017 The "MysteriumNetwork/node" Authors.
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */

package service

import (
	"fmt"
	"os"
	"strings"

	"github.com/mysteriumnetwork/node/cmd"
	"github.com/mysteriumnetwork/node/cmd/commands/license"
	"github.com/mysteriumnetwork/node/core/service"
	"github.com/mysteriumnetwork/node/metadata"
	service_noop "github.com/mysteriumnetwork/node/services/noop"
	service_openvpn "github.com/mysteriumnetwork/node/services/openvpn"
	openvpn_service "github.com/mysteriumnetwork/node/services/openvpn/service"
	service_wireguard "github.com/mysteriumnetwork/node/services/wireguard"
	"github.com/urfave/cli"
)

const serviceCommandName = "service"

var (
	identityFlag = cli.StringFlag{
		Name:  "identity",
		Usage: "Keystore's identity used to provide service. If not given identity will be created automatically",
		Value: "",
	}
	identityPassphraseFlag = cli.StringFlag{
		Name:  "identity.passphrase",
		Usage: "Used to unlock keystore's identity",
		Value: "",
	}

	agreedTermsConditionsFlag = cli.BoolFlag{
		Name:  "agreed-terms-and-conditions",
		Usage: "Agree with terms & conditions",
	}
)

// NewCommand function creates service command
func NewCommand(licenseCommandName string) *cli.Command {
	var di cmd.Dependencies
	command := &cli.Command{
		Name:      serviceCommandName,
		Usage:     "Starts and publishes services on Mysterium Network",
		ArgsUsage: "comma separated list of services to start",
		Action: func(ctx *cli.Context) error {
			if !ctx.Bool(agreedTermsConditionsFlag.Name) {
				printTermWarning(licenseCommandName)
				os.Exit(2)
			}

			nodeOptions := cmd.ParseFlagsNode(ctx)
			if err := di.Bootstrap(nodeOptions); err != nil {
				return err
			}
			if err := di.BootstrapServices(nodeOptions); err != nil {
				return err
			}

			serviceTypes := serviceTypesEnabled
			arg := ctx.Args().Get(0)
			if arg != "" {
				serviceTypes = strings.Split(arg, ",")
			}

			return runServices(ctx, &di, serviceTypes)
		},
		After: func(ctx *cli.Context) error {
			return di.Shutdown()
		},
	}

	registerFlags(&command.Flags)

	return command
}

func runServices(ctx *cli.Context, di *cmd.Dependencies, serviceTypes []string) error {
	// We need a small buffer for the error channel as we'll have quite a few concurrent reporters
	// The buffer size is determined as follows:
	// 1 for the signal callback
	// 1 for the node.Wait()
	// 1 for each of the services
	errorChannel := make(chan error, 2+len(serviceTypes))

	go func() { errorChannel <- di.Node.Wait() }()

	for _, serviceType := range serviceTypes {
		options, err := parseFlagsByServiceType(ctx, serviceType)
		if err != nil {
			return err
		}
		go func() { errorChannel <- di.ServiceManager.Start(options) }()
	}

	cmd.RegisterSignalCallback(func() { errorChannel <- nil })

	err := <-errorChannel
	switch err {
	case service.ErrorLocation:
		printLocationWarning("myst")
		return nil
	default:
		return err
	}
}

// registerFlags function register service flags to flag list
func registerFlags(flags *[]cli.Flag) {
	*flags = append(*flags,
		agreedTermsConditionsFlag,
		identityFlag, identityPassphraseFlag,
	)
	openvpn_service.RegisterFlags(flags)
}

func parseFlagsByServiceType(ctx *cli.Context, serviceType string) (service.Options, error) {
	if f, ok := serviceTypesFlagsParser[serviceType]; ok {
		return f(ctx), nil
	}
	return service.Options{}, fmt.Errorf("Unknown service type: %q", serviceType)
}

// parseOpenvpnFlags function fills in openvpn options from CLI context
func parseOpenvpnFlags(ctx *cli.Context) service.Options {
	return service.Options{
		Identity:   ctx.String(identityFlag.Name),
		Passphrase: ctx.String(identityPassphraseFlag.Name),
		Type:       service_openvpn.ServiceType,
		Options:    openvpn_service.ParseFlags(ctx),
	}
}

// parseNoopFlags function fills in noop service options from CLI context
func parseNoopFlags(ctx *cli.Context) service.Options {
	return service.Options{
		Identity:   ctx.String(identityFlag.Name),
		Passphrase: ctx.String(identityPassphraseFlag.Name),
		Type:       service_noop.ServiceType,
	}
}

// parseWireguardFlags function fills in wireguard service options from CLI context
func parseWireguardFlags(_ *cli.Context) service.Options {
	return service.Options{
		Type: service_wireguard.ServiceType,
	}
}

func printTermWarning(licenseCommandName string) {
	fmt.Println(metadata.VersionAsSummary(metadata.LicenseCopyright(
		"run program with 'myst "+licenseCommandName+" --"+license.LicenseWarrantyFlag.Name+"' option",
		"run program with 'myst "+licenseCommandName+" --"+license.LicenseConditionsFlag.Name+"' option",
	)))
	fmt.Println()

	fmt.Println("If you agree with these Terms & Conditions, run program again with '--agreed-terms-and-conditions' flag")
}

func printLocationWarning(executableName string) {
	fmt.Printf(
		"Automatic location detection failed. Enter country manually by running program again with '%s %s --%s=US' flag",
		executableName,
		serviceCommandName,
		cmd.LocationCountryFlag.Name,
	)
	fmt.Println()
}
