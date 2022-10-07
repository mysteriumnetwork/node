/*
 * Copyright (C) 2020 The "MysteriumNetwork/node" Authors.
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

package connection

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/urfave/cli/v2"

	"github.com/mysteriumnetwork/node/cmd/commands/cli/clio"
	"github.com/mysteriumnetwork/node/config"
	"github.com/mysteriumnetwork/node/config/remote"
	"github.com/mysteriumnetwork/node/core/connection"
	"github.com/mysteriumnetwork/node/core/connection/connectionstate"
	"github.com/mysteriumnetwork/node/datasize"
	"github.com/mysteriumnetwork/node/identity/registry"
	"github.com/mysteriumnetwork/node/money"
	tequilapi_client "github.com/mysteriumnetwork/node/tequilapi/client"
	"github.com/mysteriumnetwork/node/tequilapi/contract"
	"github.com/mysteriumnetwork/terms/terms-go"
)

// CommandName is the name of this command
const CommandName = "connection"

var (
	flagProxyPort = cli.IntFlag{
		Name:  "proxy",
		Usage: "Proxy port",
	}

	flagCountry = cli.StringFlag{
		Name:  "country",
		Usage: "Two letter (ISO 3166-1 alpha-2) country code to filter proposals.",
	}

	flagLocationType = cli.StringFlag{
		Name:  "location-type",
		Usage: "Node location types to filter by eg.'hosting', 'residential', 'mobile' etc.",
	}

	flagSortType = cli.StringFlag{
		Name:  "sort",
		Usage: "Proposal sorting type. One of: quality, bandwidth, latency, uptime or price",
		Value: "quality",
	}

	flagIncludeFailed = cli.BoolFlag{
		Name:  "include-failed",
		Usage: "Include proposals marked as test failed by monitoring agent",
		Value: false,
	}
)

const serviceWireguard = "wireguard"

// NewCommand function creates license command.
func NewCommand() *cli.Command {
	var cmd *command

	return &cli.Command{
		Name:        CommandName,
		Usage:       "Manage your connection",
		Description: "Using the connection subcommands you can manage your connection or get additional information about it",
		Flags:       []cli.Flag{&config.FlagTequilapiAddress, &config.FlagTequilapiPort},
		Before: func(ctx *cli.Context) error {
			tc, err := clio.NewTequilApiClient(ctx)
			if err != nil {
				return err
			}

			cfg, err := remote.NewConfig(tc)
			if err != nil {
				return err
			}

			cmd = &command{
				tequilapi: tc,
				cfg:       cfg,
			}
			return nil
		},
		Subcommands: []*cli.Command{
			{
				Name:  "proposals",
				Usage: "List all possible proposals to which you can connect",
				Flags: []cli.Flag{&flagCountry, &flagLocationType},
				Action: func(ctx *cli.Context) error {
					cmd.proposals(ctx)
					return nil
				},
			},
			{
				Name:      "up",
				ArgsUsage: "[ProviderIdentityAddress]",
				Usage:     "Create a new connection",
				Flags:     []cli.Flag{&config.FlagAgreedTermsConditions, &flagCountry, &flagLocationType, &flagSortType, &flagIncludeFailed, &flagProxyPort},
				Action: func(ctx *cli.Context) error {
					cmd.up(ctx)
					return nil
				},
			},
			{
				Name:  "down",
				Usage: "Disconnect from your current connection",
				Flags: []cli.Flag{&flagProxyPort},
				Action: func(ctx *cli.Context) error {
					cmd.down(ctx)
					return nil
				},
			},
			{
				Name:  "info",
				Usage: "Show information about your connection",
				Flags: []cli.Flag{&flagProxyPort},
				Action: func(ctx *cli.Context) error {
					cmd.info(ctx)
					return nil
				},
			},
		},
	}
}

type command struct {
	tequilapi *tequilapi_client.Client
	cfg       *remote.Config
}

func (c *command) proposals(ctx *cli.Context) {
	locationType := ctx.String(flagLocationType.Name)
	locationCountry := ctx.String(flagCountry.Name)
	if locationCountry != "" && len(locationCountry) != 2 {
		clio.Warn("Country code must be in ISO 3166-1 alpha-2 format. Example: 'UK', 'US'")
		return
	}

	proposals, err := c.tequilapi.ProposalsByLocationAndService(serviceWireguard, locationType, locationCountry)
	if err != nil {
		clio.Warn("Failed to fetch proposal list")
		return
	}

	if len(proposals) == 0 {
		clio.Info("No proposals found")
		return
	}

	clio.Info("Found proposals:")
	w := tabwriter.NewWriter(os.Stdout, 1, 1, 1, ' ', 0)
	for _, p := range proposals {
		fmt.Fprintln(w, proposalFormatted(&p))
	}
	w.Flush()
}

func (c *command) down(ctx *cli.Context) {
	status, err := c.tequilapi.ConnectionStatus(ctx.Int(flagProxyPort.Name))
	if err != nil {
		clio.Warn("Could not get connection status")
		return
	}

	if status.Status != string(connectionstate.NotConnected) {
		if err := c.tequilapi.ConnectionDestroy(ctx.Int(flagProxyPort.Name)); err != nil {
			clio.Warn(err)
			return
		}
	}

	clio.Success("Disconnected")
}

func (c *command) handleTOS(ctx *cli.Context) error {
	if ctx.Bool(config.FlagAgreedTermsConditions.Name) {
		c.acceptTOS()
		return nil
	}

	agreed := c.cfg.GetBool(contract.TermsConsumerAgreed)
	if !agreed {
		return errors.New("you must agree with consumer terms of use in order to use this command")
	}

	version := c.cfg.GetString(contract.TermsVersion)
	if version != terms.TermsVersion {
		return fmt.Errorf("you've agreed to terms of use version %s, but version %s is required", version, terms.TermsVersion)
	}

	return nil
}

func (c *command) acceptTOS() {
	t := true
	if err := c.tequilapi.UpdateTerms(contract.TermsRequest{
		AgreedConsumer: &t,
		AgreedVersion:  terms.TermsVersion,
	}); err != nil {
		clio.Info("Failed to save terms of use agreement, you will have to re-agree on next launch")
	}
}

func (c *command) up(ctx *cli.Context) {
	if err := c.handleTOS(ctx); err != nil {
		clio.PrintTOSError(err)
		return
	}

	status, err := c.tequilapi.ConnectionStatus(ctx.Int(flagProxyPort.Name))
	if err != nil {
		clio.Warn("Could not get connection status")
		return
	}

	switch connectionstate.State(status.Status) {
	case
		// connectionstate.Connected,
		connectionstate.Connecting,
		connectionstate.Disconnecting,
		connectionstate.Reconnecting:

		msg := fmt.Sprintf("You can't create a new connection, you're in state '%s'", status.Status)
		clio.Warn(msg)
		return
	}

	providers := strings.Split(ctx.Args().First(), ",")
	providerIDs := []string{}

	for _, p := range providers {
		if len(p) > 0 {
			providerIDs = append(providerIDs, p)
		}
	}

	id, err := c.tequilapi.CurrentIdentity("", "")
	if err != nil {
		clio.Error("Failed to get your identity")
		return
	}

	identityStatus, err := c.tequilapi.Identity(id.Address)
	if err != nil {
		clio.Warn("Failed to get identity status")
		return
	}

	if identityStatus.RegistrationStatus != registry.Registered.String() {
		clio.Warn("Your identity is not registered, please execute `myst account register` first")
		return
	}

	clio.Status("CONNECTING", "Creating connection from:", id.Address, "to:", providers)

	connectOptions := contract.ConnectOptions{
		DNS:               connection.DNSOptionAuto,
		DisableKillSwitch: false,
		ProxyPort:         ctx.Int(flagProxyPort.Name),
	}
	hermesID, err := c.cfg.GetHermesID()
	if err != nil {
		clio.Error(err)
		return
	}

	filter := contract.ConnectionCreateFilter{
		Providers:               providerIDs,
		CountryCode:             ctx.String(flagCountry.Name),
		IPType:                  ctx.String(flagLocationType.Name),
		SortBy:                  ctx.String(flagSortType.Name),
		IncludeMonitoringFailed: ctx.Bool(flagIncludeFailed.Name),
	}

	_, err = c.tequilapi.SmartConnectionCreate(id.Address, hermesID, serviceWireguard, filter, connectOptions)
	if err != nil {
		clio.Error("Failed to create a new connection: ", err)
		return
	}

	clio.Success("Connected")
}

func (c *command) info(ctx *cli.Context) {
	inf := newConnInfo()

	id, err := c.tequilapi.CurrentIdentity("", "")
	if err == nil {
		inf.set(infIdentity, id.Address)
	}

	status, err := c.tequilapi.ConnectionStatus(ctx.Int(flagProxyPort.Name))
	if err == nil {
		if status.Status == string(connectionstate.Connected) {
			inf.isConnected = true
			inf.set(infProposal, status.Proposal.String())
		}

		inf.set(infStatus, status.Status)
		inf.set(infSessionID, status.SessionID)
	}

	ip, err := c.tequilapi.ProxyIP(ctx.Int(flagProxyPort.Name))
	if err == nil {
		inf.set(infIP, ip.IP)
	}

	location, err := c.tequilapi.ProxyLocation(ctx.Int(flagProxyPort.Name))
	if err == nil {
		inf.set(infLocation, fmt.Sprintf("%s, %s (%s - %s)", location.City, location.Country, location.IPType, location.ISP))
	}

	if status.Status != string(connectionstate.Connected) {
		inf.printAll()
		return
	}

	statistics, err := c.tequilapi.ConnectionStatistics(status.SessionID)
	if err == nil {
		inf.set(infDuration, fmt.Sprint(time.Duration(statistics.Duration)*time.Second))
		inf.set(infTransferred, fmt.Sprintf("%s/%s", datasize.FromBytes(statistics.BytesReceived), datasize.FromBytes(statistics.BytesSent)))
		inf.set(infThroughput, fmt.Sprintf("%s/%s", datasize.BitSpeed(statistics.ThroughputReceived), datasize.BitSpeed(statistics.ThroughputSent)))
		inf.set(infSpent, money.New(statistics.TokensSpent).String())
	}

	inf.printAll()
}
