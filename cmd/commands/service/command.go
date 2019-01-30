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
	"github.com/mysteriumnetwork/node/core/node"
	"github.com/mysteriumnetwork/node/core/service"
	"github.com/mysteriumnetwork/node/identity"
	identity_selector "github.com/mysteriumnetwork/node/identity/selector"
	"github.com/mysteriumnetwork/node/metadata"
	openvpn_service "github.com/mysteriumnetwork/node/services/openvpn/service"
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

			providerID, err := unlockIdentity(ctx, &di, nodeOptions)
			if err != nil {
				return err
			}

			return runServices(ctx, &di, providerID)
		},
		After: func(ctx *cli.Context) error {
			return di.Shutdown()
		},
	}

	registerFlags(&command.Flags)

	return command
}

func unlockIdentity(ctx *cli.Context, di *cmd.Dependencies, nodeOptions node.Options) (identity.Identity, error) {
	identityOptions := parseFlags(ctx)

	identityHandler := identity_selector.NewHandler(
		di.IdentityManager,
		di.MysteriumAPI,
		identity.NewIdentityCache(nodeOptions.Directories.Keystore, "remember.json"),
		di.SignerFactory,
	)
	loadIdentity := identity_selector.NewLoader(identityHandler, identityOptions.Identity, identityOptions.Passphrase)

	return loadIdentity()
}

func runServices(ctx *cli.Context, di *cmd.Dependencies, providerID identity.Identity) error {
	serviceTypes := serviceTypesEnabled
	arg := ctx.Args().Get(0)
	if arg != "" {
		serviceTypes = strings.Split(arg, ",")
	}

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
		go func() { errorChannel <- di.ServiceManager.Start(providerID, serviceType, options) }()
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

// parseFlags function fills in service command options from CLI context
func parseFlags(ctx *cli.Context) service.OptionsIdentity {
	return service.OptionsIdentity{
		Identity:   ctx.String(identityFlag.Name),
		Passphrase: ctx.String(identityPassphraseFlag.Name),
	}
}

func parseFlagsByServiceType(ctx *cli.Context, serviceType string) (service.Options, error) {
	if f, ok := serviceTypesFlagsParser[serviceType]; ok {
		return f(ctx), nil
	}
	return service.OptionsIdentity{}, fmt.Errorf("unknown service type: %q", serviceType)
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
