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
	"time"

	"github.com/mysteriumnetwork/node/cmd"
	"github.com/mysteriumnetwork/node/config"
	"github.com/mysteriumnetwork/node/config/urfavecli/clicontext"
	"github.com/mysteriumnetwork/node/core/node"
	"github.com/mysteriumnetwork/node/core/service"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/metadata"
	"github.com/mysteriumnetwork/node/services"
	"github.com/mysteriumnetwork/node/tequilapi/client"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"
)

// NewCommand function creates service command
func NewCommand(licenseCommandName string) *cli.Command {
	var di cmd.Dependencies
	command := &cli.Command{
		Name:      "service",
		Usage:     "Starts and publishes services on Mysterium Network",
		ArgsUsage: "comma separated list of services to start",
		Before:    clicontext.LoadUserConfigQuietly,
		Action: func(ctx *cli.Context) error {
			if !ctx.Bool(config.FlagAgreedTermsConditions.Name) {
				printTermWarning(licenseCommandName)
				os.Exit(2)
			}

			quit := make(chan error)
			config.ParseFlagsNode(ctx)
			config.ParseFlagsServiceShared(ctx)
			nodeOptions := node.GetOptions()
			if err := di.Bootstrap(*nodeOptions); err != nil {
				return err
			}
			go func() { quit <- di.Node.Wait() }()

			cmd.RegisterSignalCallback(func() { quit <- nil })

			cmdService := &serviceCommand{
				tequilapi:    client.NewClient(nodeOptions.TequilapiAddress, nodeOptions.TequilapiPort),
				errorChannel: quit,
				ap: client.AccessPoliciesRequest{
					IDs: services.SharedConfiguredOptions().AccessPolicyList,
				},
			}

			go func() {
				quit <- cmdService.Run(ctx)
			}()

			return describeQuit(<-quit)
		},
		After: func(ctx *cli.Context) error {
			return di.Shutdown()
		},
	}

	registerFlags(&command.Flags)

	return command
}

func describeQuit(err error) error {
	if err == nil {
		log.Info().Msg("Stopping application")
	} else {
		log.Error().Err(err).Stack().Msg("Terminating application due to error")
	}
	return err
}

// serviceCommand represent entrypoint for service command with top level components
type serviceCommand struct {
	tequilapi    *client.Client
	errorChannel chan error
	ap           client.AccessPoliciesRequest
}

// Run runs a command
func (sc *serviceCommand) Run(ctx *cli.Context) (err error) {
	arg := ctx.Args().Get(0)
	if arg != "" {
		serviceTypes = strings.Split(arg, ",")
	}

	providerID := sc.unlockIdentity(parseIdentityFlags(ctx))
	log.Info().Msgf("Unlocked identity: %v", providerID.Address)

	if err := sc.runServices(ctx, providerID.Address, serviceTypes); err != nil {
		return err
	}

	return <-sc.errorChannel
}

func (sc *serviceCommand) unlockIdentity(identityOptions service.OptionsIdentity) *identity.Identity {
	const retryRate = 10 * time.Second
	for {
		id, err := sc.tequilapi.CurrentIdentity(identityOptions.Identity, identityOptions.Passphrase)
		if err == nil {
			return &identity.Identity{Address: id.Address}
		}
		log.Warn().Err(err).Msg("Failed to get current identity")
		log.Warn().Msgf("retrying in %vs...", retryRate.Seconds())
		time.Sleep(retryRate)
	}
}

func (sc *serviceCommand) runServices(ctx *cli.Context, providerID string, serviceTypes []string) error {
	for _, serviceType := range serviceTypes {
		options, err := parseFlagsByServiceType(ctx, serviceType)
		if err != nil {
			return err
		}
		go sc.runService(providerID, serviceType, options)
	}

	return nil
}

func (sc *serviceCommand) runService(providerID, serviceType string, options service.Options) {
	_, err := sc.tequilapi.ServiceStart(providerID, serviceType, options, sc.ap)
	if err != nil {
		sc.errorChannel <- errors.Wrapf(err, "failed to run service %s", serviceType)
	}
}

// registerFlags function register service flags to flag list
func registerFlags(flags *[]cli.Flag) {
	config.RegisterFlagsServiceShared(flags)
	config.RegisterFlagsServiceOpenvpn(flags)
	config.RegisterFlagsServiceWireguard(flags)
}

// parseIdentityFlags function fills in service command options from CLI context
func parseIdentityFlags(ctx *cli.Context) service.OptionsIdentity {
	return service.OptionsIdentity{
		Identity:   ctx.String(config.FlagIdentity.Name),
		Passphrase: ctx.String(config.FlagIdentityPassphrase.Name),
	}
}

func parseFlagsByServiceType(ctx *cli.Context, serviceType string) (service.Options, error) {
	if f, ok := serviceTypesFlagsParser[serviceType]; ok {
		return f(ctx), nil
	}
	return service.OptionsIdentity{}, errors.Errorf("unknown service type: %q", serviceType)
}

func printTermWarning(licenseCommandName string) {
	fmt.Println(metadata.VersionAsSummary(metadata.LicenseCopyright(
		"run program with 'myst "+licenseCommandName+" --"+config.LicenseWarrantyFlag.Name+"' option",
		"run program with 'myst "+licenseCommandName+" --"+config.LicenseConditionsFlag.Name+"' option",
	)))
	fmt.Println()

	fmt.Println("If you agree with these Terms & Conditions, run program again with '--agreed-terms-and-conditions' flag")
}
