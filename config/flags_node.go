/*
 * Copyright (C) 2019 The "MysteriumNetwork/node" Authors.
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

package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"
)

var (
	// Alphabetically sorted list of node flags
	// Some of the flags are location in separate source files: flags_*.go

	// FlagDiscoveryType proposal discovery adapter.
	FlagDiscoveryType = cli.StringSliceFlag{
		Name:  "discovery.type",
		Usage: `Proposal discovery adapter(s) separated by comma Options: { "api", "broker", "api,broker" }`,
		Value: cli.NewStringSlice("api", "broker"),
	}
	// FlagDiscoveryPingInterval proposal ping interval in seconds.
	FlagDiscoveryPingInterval = cli.DurationFlag{
		Name:  "discovery.ping",
		Usage: `Proposal update interval { "30s", "3m", "1h20m30s" }`,
		Value: 180 * time.Second,
	}
	// FlagDiscoveryFetchInterval proposal fetch interval in seconds.
	FlagDiscoveryFetchInterval = cli.DurationFlag{
		Name:  "discovery.fetch",
		Usage: `Proposal fetch interval { "30s", "3m", "1h20m30s" }`,
		Value: 180 * time.Second,
	}
	// FlagBindAddress IP address to bind to.
	FlagBindAddress = cli.StringFlag{
		Name:  "bind.address",
		Usage: "IP address to bind provided services to",
		Value: "0.0.0.0",
	}
	// FlagFeedbackURL URL of Feedback API.
	FlagFeedbackURL = cli.StringFlag{
		Name:  "feedback.url",
		Usage: "URL of Feedback API",
		Value: "https://feedback.mysterium.network",
	}
	// FlagFirewallKillSwitch always blocks non-tunneled outgoing consumer traffic.
	FlagFirewallKillSwitch = cli.BoolFlag{
		Name:  "firewall.killSwitch.always",
		Usage: "Always block non-tunneled outgoing consumer traffic",
	}
	// FlagFirewallProtectedNetworks protects provider's networks from access via VPN
	FlagFirewallProtectedNetworks = cli.StringFlag{
		Name:  "firewall.protected.networks",
		Usage: "List of comma separated (no spaces) subnets to be protected from access via VPN",
		Value: "10.0.0.0/8,172.16.0.0/12,192.168.0.0/16,127.0.0.0/8",
	}
	// FlagShaperEnabled enables bandwidth limitation.
	FlagShaperEnabled = cli.BoolFlag{
		Name:  "shaper.enabled",
		Usage: "Limit service bandwidth",
	}
	// FlagKeystoreLightweight determines the scrypt memory complexity.
	FlagKeystoreLightweight = cli.BoolFlag{
		Name:  "keystore.lightweight",
		Usage: "Determines the scrypt memory complexity. If set to true, will use 4MB blocks instead of the standard 256MB ones",
		Value: true,
	}
	// FlagLogHTTP enables HTTP payload logging.
	FlagLogHTTP = cli.BoolFlag{
		Name:  "log.http",
		Usage: "Enable HTTP payload logging",
	}
	// FlagLogLevel logger level.
	FlagLogLevel = cli.StringFlag{
		Name: "log-level",
		Usage: func() string {
			allLevels := []string{
				zerolog.TraceLevel.String(),
				zerolog.DebugLevel.String(),
				zerolog.InfoLevel.String(),
				zerolog.WarnLevel.String(),
				zerolog.FatalLevel.String(),
				zerolog.PanicLevel.String(),
				zerolog.Disabled.String(),
			}
			return fmt.Sprintf("Set the logging level (%s)", strings.Join(allLevels, "|"))
		}(),
		Value: zerolog.DebugLevel.String(),
	}
	// FlagOpenvpnBinary openvpn binary to use for OpenVPN connections.
	FlagOpenvpnBinary = cli.StringFlag{
		Name:  "openvpn.binary",
		Usage: "openvpn binary to use for OpenVPN connections",
		Value: "openvpn",
	}
	// FlagQualityType quality oracle adapter.
	FlagQualityType = cli.StringFlag{
		Name:  "quality.type",
		Usage: "Quality Oracle adapter. Options:  (elastic, morqa, none - opt-out from sending quality metrics)",
		Value: "morqa",
	}
	// FlagQualityAddress quality oracle URL.
	FlagQualityAddress = cli.StringFlag{
		Name: "quality.address",
		Usage: fmt.Sprintf(
			"Address of specific Quality Oracle adapter given in '--%s'",
			FlagQualityType.Name,
		),
		Value: "https://quality.mysterium.network/api/v1",
	}
	// FlagTequilapiAddress IP address of interface to listen for incoming connections.
	FlagTequilapiAddress = cli.StringFlag{
		Name:  "tequilapi.address",
		Usage: "IP address to bind Tequilapi to",
		Value: "127.0.0.1",
	}
	// FlagTequilapiPort port for listening for incoming API requests.
	FlagTequilapiPort = cli.IntFlag{
		Name:  "tequilapi.port",
		Usage: "Port for listening incoming api requests",
		Value: 4050,
	}
	// FlagPProfEnable enables pprof via TequilAPI.
	FlagPProfEnable = cli.BoolFlag{
		Name:  "pprof.enable",
		Usage: "Enables pprof",
		Value: false,
	}
	// FlagUIEnable enables built-in web UI for node.
	FlagUIEnable = cli.BoolFlag{
		Name:  "ui.enable",
		Usage: "Enables the Web UI",
		Value: true,
	}
	// FlagUIAddress IP address of interface to listen for incoming connections.
	FlagUIAddress = cli.StringFlag{
		Name:  "ui.address",
		Usage: "IP address to bind Web UI to (by default, local LAN IP)",
		Value: "",
	}
	// FlagUIPort runs web UI on the specified port.
	FlagUIPort = cli.IntFlag{
		Name:  "ui.port",
		Usage: "The port to run Web UI on",
		Value: 4449,
	}
	// FlagUserMode allows to run node under current user without sudo.
	FlagUserMode = cli.BoolFlag{
		Name:  "usermode",
		Usage: "Run as a regular user. Delegate elevated commands to the supervisor.",
		Value: false,
	}
	// FlagVendorID identifies 3rd party vendor (distributor) of Mysterium node.
	FlagVendorID = cli.StringFlag{
		Name: "vendor.id",
		Usage: "Marks vendor (distributor) of the node for collecting statistics. " +
			"3rd party vendors may use their own identifier here.",
	}
	//FlagP2PListenPorts sets manual ports for p2p connections.
	FlagP2PListenPorts = cli.StringFlag{
		Name:  "p2p.listen.ports",
		Usage: "Range of P2P listen ports (e.g. 51820:52075), value of 0:0 means disabled",
		Value: "0:0",
	}

	//FlagConsumer sets to run as consumer only which allows to skip bootstrap for some of the dependencies.
	FlagConsumer = cli.BoolFlag{
		Name:  "consumer",
		Usage: "Run in consumer mode only.",
		Value: false,
	}
)

// RegisterFlagsNode function register node flags to flag list
func RegisterFlagsNode(flags *[]cli.Flag) error {
	if err := RegisterFlagsDirectory(flags); err != nil {
		return err
	}

	RegisterFlagsLocation(flags)
	RegisterFlagsNetwork(flags)
	RegisterFlagsTransactor(flags)
	RegisterFlagsHermes(flags)
	RegisterFlagsPayments(flags)
	RegisterFlagsPolicy(flags)
	RegisterFlagsMMN(flags)

	*flags = append(*flags,
		&FlagBindAddress,
		&FlagDiscoveryType,
		&FlagDiscoveryPingInterval,
		&FlagDiscoveryFetchInterval,
		&FlagFeedbackURL,
		&FlagFirewallKillSwitch,
		&FlagFirewallProtectedNetworks,
		&FlagShaperEnabled,
		&FlagKeystoreLightweight,
		&FlagLogHTTP,
		&FlagLogLevel,
		&FlagOpenvpnBinary,
		&FlagQualityType,
		&FlagQualityAddress,
		&FlagTequilapiAddress,
		&FlagTequilapiPort,
		&FlagPProfEnable,
		&FlagUIEnable,
		&FlagUIAddress,
		&FlagUIPort,
		&FlagUserMode,
		&FlagVendorID,
		&FlagP2PListenPorts,
		&FlagConsumer,
	)

	return nil
}

// ParseFlagsNode function fills in node options from CLI context
func ParseFlagsNode(ctx *cli.Context) {
	ParseFlagsDirectory(ctx)

	ParseFlagsLocation(ctx)
	ParseFlagsNetwork(ctx)
	ParseFlagsTransactor(ctx)
	ParseFlagsHermes(ctx)
	ParseFlagsPayments(ctx)
	ParseFlagsPolicy(ctx)
	ParseFlagsMMN(ctx)

	Current.ParseStringFlag(ctx, FlagBindAddress)
	Current.ParseStringSliceFlag(ctx, FlagDiscoveryType)
	Current.ParseDurationFlag(ctx, FlagDiscoveryPingInterval)
	Current.ParseDurationFlag(ctx, FlagDiscoveryFetchInterval)
	Current.ParseStringFlag(ctx, FlagFeedbackURL)
	Current.ParseBoolFlag(ctx, FlagFirewallKillSwitch)
	Current.ParseStringFlag(ctx, FlagFirewallProtectedNetworks)
	Current.ParseBoolFlag(ctx, FlagShaperEnabled)
	Current.ParseBoolFlag(ctx, FlagKeystoreLightweight)
	Current.ParseBoolFlag(ctx, FlagLogHTTP)
	Current.ParseStringFlag(ctx, FlagLogLevel)
	Current.ParseStringFlag(ctx, FlagOpenvpnBinary)
	Current.ParseStringFlag(ctx, FlagQualityAddress)
	Current.ParseStringFlag(ctx, FlagQualityType)
	Current.ParseStringFlag(ctx, FlagTequilapiAddress)
	Current.ParseIntFlag(ctx, FlagTequilapiPort)
	Current.ParseBoolFlag(ctx, FlagPProfEnable)
	Current.ParseBoolFlag(ctx, FlagUIEnable)
	Current.ParseStringFlag(ctx, FlagUIAddress)
	Current.ParseIntFlag(ctx, FlagUIPort)
	Current.ParseBoolFlag(ctx, FlagUserMode)
	Current.ParseStringFlag(ctx, FlagVendorID)
	Current.ParseStringFlag(ctx, FlagP2PListenPorts)
	Current.ParseBoolFlag(ctx, FlagConsumer)

	ValidateAddressFlags(FlagTequilapiAddress)
}

// ValidateAddressFlags validates given address flags for public exposure
func ValidateAddressFlags(flags ...cli.StringFlag) {
	for _, flag := range flags {
		if flag.Value == "localhost" || flag.Value == "127.0.0.1" {
			return
		}
		log.Warn().Msgf("Possible security vulnerability by flag `%s`, `%s` might be reachable from outside! "+
			"Ensure its set to localhost or protected by firewall.", flag.Name, flag.Value)
	}
}
