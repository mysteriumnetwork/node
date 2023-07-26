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

	"github.com/mysteriumnetwork/terms/terms-go"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"

	"github.com/mysteriumnetwork/node/cmd"
	"github.com/mysteriumnetwork/node/cmd/commands/cli/clio"
	"github.com/mysteriumnetwork/node/config"
	"github.com/mysteriumnetwork/node/config/urfavecli/clicontext"
	"github.com/mysteriumnetwork/node/core/node"
	"github.com/mysteriumnetwork/node/metadata"
	"github.com/mysteriumnetwork/node/services"
	"github.com/mysteriumnetwork/node/tequilapi/client"
	"github.com/mysteriumnetwork/node/tequilapi/contract"
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
			quit := make(chan error)
			config.ParseFlagsServiceStart(ctx)
			config.ParseFlagsServiceOpenvpn(ctx)
			config.ParseFlagsServiceWireguard(ctx)
			config.ParseFlagsServiceNoop(ctx)
			config.ParseFlagsNode(ctx)

			if err := hasAcceptedTOS(ctx); err != nil {
				clio.PrintTOSError(err)
				os.Exit(2)
			}

			nodeOptions := node.GetOptions()
			nodeOptions.Discovery.FetchEnabled = false
			if err := di.Bootstrap(*nodeOptions); err != nil {
				return err
			}
			go func() { quit <- di.Node.Wait() }()

			cmd.RegisterSignalCallback(func() { quit <- nil })

			cmdService := &serviceCommand{
				tequilapi:    client.NewClient(nodeOptions.TequilapiAddress, nodeOptions.TequilapiPort),
				errorChannel: quit,
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

	config.RegisterFlagsServiceStart(&command.Flags)
	config.RegisterFlagsServiceOpenvpn(&command.Flags)
	config.RegisterFlagsServiceWireguard(&command.Flags)
	config.RegisterFlagsServiceNoop(&command.Flags)

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
}

// Run runs a command
func (sc *serviceCommand) Run(ctx *cli.Context) (err error) {
	serviceTypes := make([]string, 0)

	activeServices := config.Current.GetString(config.FlagActiveServices.Name)
	if len(activeServices) != 0 {
		serviceTypes = strings.Split(activeServices, ",")
	}

	sc.tryRememberTOS(ctx, sc.errorChannel)
	providerID := sc.unlockIdentity(
		ctx.String(config.FlagIdentity.Name),
		ctx.String(config.FlagIdentityPassphrase.Name),
	)
	log.Info().Msgf("Unlocked identity: %v", providerID)

	if config.Current.GetString(config.FlagNodeVersion.Name) == "" {

		// on first version update: enable dvpn service if wireguard service is enabled
		mapServices := make(map[string]bool, 0)
		for _, serviceType := range serviceTypes {
			mapServices[serviceType] = true
		}

		if mapServices["wireguard"] && !mapServices["dvpn"] {
			serviceTypes = append(serviceTypes, "dvpn")
		}
	}
	// save the version
	config.Current.SetUser(config.FlagNodeVersion.Name, metadata.BuildNumber)
	config.Current.SaveUserConfig()

	for _, serviceType := range serviceTypes {
		serviceOpts, err := services.GetStartOptions(serviceType)
		if err != nil {
			return err
		}
		startRequest := contract.ServiceStartRequest{
			ProviderID:     providerID,
			Type:           serviceType,
			AccessPolicies: &contract.ServiceAccessPolicies{IDs: serviceOpts.AccessPolicyList},
			Options:        serviceOpts,
		}

		go sc.runService(startRequest)
	}

	return <-sc.errorChannel
}

func (sc *serviceCommand) unlockIdentity(id, passphrase string) string {
	const retryRate = 10 * time.Second
	for {
		id, err := sc.tequilapi.CurrentIdentity(id, passphrase)
		if err == nil {
			return id.Address
		}
		log.Warn().Err(err).Msg("Failed to get current identity")
		log.Warn().Msgf("retrying in %vs...", retryRate.Seconds())
		time.Sleep(retryRate)
	}
}

func (sc *serviceCommand) tryRememberTOS(ctx *cli.Context, errCh chan error) {
	if !ctx.Bool(config.FlagAgreedTermsConditions.Name) {
		return
	}

	doUpdate := func() {
		t := true
		for i := 0; i < 5; i++ {
			if err := sc.tequilapi.UpdateTerms(contract.TermsRequest{
				AgreedProvider: &t,
				AgreedConsumer: &t,
				AgreedVersion:  terms.TermsVersion,
			}); err == nil {
				return
			}
			time.Sleep(time.Second * 2)
		}
	}

	go func() {
		select {
		case <-errCh:
			return
		default:
			doUpdate()
		}
	}()
}

func (sc *serviceCommand) runService(request contract.ServiceStartRequest) {
	_, err := sc.tequilapi.ServiceStart(request)
	if err != nil {
		sc.errorChannel <- errors.Wrapf(err, "failed to run service %s", request.Type)
	}
}

func hasAcceptedTOS(ctx *cli.Context) error {
	if ctx.Bool(config.FlagAgreedTermsConditions.Name) {
		return nil
	}

	agreed := config.Current.GetBool(contract.TermsProviderAgreed)
	if !agreed {
		return errors.New("You must agree with provider terms of use in order to use this command")
	}

	version := config.Current.GetString(contract.TermsVersion)
	if version != terms.TermsVersion {
		return fmt.Errorf("you've agreed to terms of use version %s, but version %s is required", version, terms.TermsVersion)
	}

	return nil
}
