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

	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"

	"github.com/mysteriumnetwork/node/metadata"
)

var (
	// Alphabetically sorted list of node flags
	// Some of the flags are location in separate source files: flags_*.go

	// FlagDiscoveryType proposal discovery adapter.
	FlagDiscoveryType = cli.StringSliceFlag{
		Name:  "discovery.type",
		Usage: `Proposal discovery adapter(s) separated by comma. Options: { "api", "broker", "api,broker,dht" }`,
		Value: cli.NewStringSlice("api"),
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
	// FlagDHTAddress IP address of interface to listen for DHT connections.
	FlagDHTAddress = cli.StringFlag{
		Name:  "discovery.dht.address",
		Usage: "IP address to bind DHT to",
		Value: "0.0.0.0",
	}
	// FlagDHTPort listens DHT connections on the specified port.
	FlagDHTPort = cli.IntFlag{
		Name:  "discovery.dht.port",
		Usage: "The port to bind DHT to (by default, random port will be used)",
		Value: 0,
	}
	// FlagDHTProtocol protocol for DHT to use.
	FlagDHTProtocol = cli.StringFlag{
		Name:  "discovery.dht.proto",
		Usage: "Protocol to use with DHT. Options: { udp, tcp }",
		Value: "tcp",
	}
	// FlagDHTBootstrapPeers DHT bootstrap peer nodes list.
	FlagDHTBootstrapPeers = cli.StringSliceFlag{
		Name:  "discovery.dht.peers",
		Usage: `Peer URL(s) for DHT bootstrap (e.g. /ip4/127.0.0.1/tcp/1234/p2p/QmNUZRp1zrk8i8TpfyeDZ9Yg3C4PjZ5o61yao3YhyY1TE8") separated by comma. They will tell us about the other nodes in the network.`,
		Value: cli.NewStringSlice(),
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
	// FlagShaperBandwidth set the bandwidth limit.
	FlagShaperBandwidth = cli.Uint64Flag{
		Name:  "shaper.bandwidth",
		Usage: "Set the bandwidth limit in Kbytes",
		Value: 6250,
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
	// FlagVerbose enables verbose logging.
	FlagVerbose = cli.BoolFlag{
		Name:  "verbose",
		Usage: "Enable verbose logging",
		Value: false,
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
		Value: "https://quality.mysterium.network/api/v3",
	}
	// FlagTequilapiAddress IP address of interface to listen for incoming connections.
	FlagTequilapiAddress = cli.StringFlag{
		Name:  "tequilapi.address",
		Usage: "IP address to bind API to",
		Value: "127.0.0.1",
	}
	// FlagTequilapiAllowedHostnames Restrict hostnames in requests' Host header to following domains.
	FlagTequilapiAllowedHostnames = cli.StringFlag{
		Name:  "tequilapi.allowed-hostnames",
		Usage: "Comma separated list of allowed domains. Prepend value with dot for wildcard mask",
		Value: ".localhost, localhost, .localdomain",
	}
	// FlagTequilapiPort port for listening for incoming API requests.
	FlagTequilapiPort = cli.IntFlag{
		Name:  "tequilapi.port",
		Usage: "Port for listening incoming API requests",
		Value: 4050,
	}
	// FlagTequilapiDebugMode debug mode for tequilapi.
	FlagTequilapiDebugMode = cli.BoolFlag{
		Name:  "tequilapi.debug",
		Usage: "Starts tequilapi in debug mode",
		Value: false,
	}
	// FlagTequilapiUsername username for API authentication.
	FlagTequilapiUsername = cli.StringFlag{
		Name:  "tequilapi.auth.username",
		Usage: "Default username for API authentication",
		Value: "myst",
	}
	// FlagTequilapiPassword username for API authentication.
	FlagTequilapiPassword = cli.StringFlag{
		Name:  "tequilapi.auth.password",
		Usage: "Default password for API authentication",
		Value: "mystberry",
	}
	// FlagPProfEnable enables pprof via TequilAPI.
	FlagPProfEnable = cli.BoolFlag{
		Name:  "pprof.enable",
		Usage: "Enables pprof",
		Value: false,
	}
	// FlagUserMode allows running node under current user without sudo.
	FlagUserMode = cli.BoolFlag{
		Name:  "usermode",
		Usage: "Run as a regular user. Delegate elevated commands to the supervisor.",
		Value: false,
	}

	// FlagDVPNMode allows running node in a kernelspace without establishing system-wite tunnels.
	FlagDVPNMode = cli.BoolFlag{
		Name:  "dvpnmode",
		Usage: "Run in a kernelspace without establishing system-wite tunnels",
		Value: false,
	}

	// FlagProxyMode allows running node under current user as a proxy.
	FlagProxyMode = cli.BoolFlag{
		Name:  "proxymode",
		Usage: "Run as a regular user as a proxy",
		Value: false,
	}

	// FlagProvCheckerMode allows running node under current user as a provider checker agent.
	FlagProvCheckerMode = cli.BoolFlag{
		Name:  "provchecker",
		Usage: "",
		Value: false,
	}

	// FlagUserspace allows running a node without privileged permissions.
	FlagUserspace = cli.BoolFlag{
		Name:  "userspace",
		Usage: "Run a node without privileged permissions",
		Value: false,
	}

	// FlagVendorID identifies 3rd party vendor (distributor) of Mysterium node.
	FlagVendorID = cli.StringFlag{
		Name: "vendor.id",
		Usage: "Marks vendor (distributor) of the node for collecting statistics. " +
			"3rd party vendors may use their own identifier here.",
	}
	// FlagLauncherVersion is used for reporting the version of a Launcher.
	FlagLauncherVersion = cli.StringFlag{
		Name:  "launcher.ver",
		Usage: "Report the version of a launcher for statistics",
	}

	// FlagP2PListenPorts sets manual ports for p2p connections.
	// TODO: remove the deprecated flag once all users stop to use it.
	FlagP2PListenPorts = cli.StringFlag{
		Name:  "p2p.listen.ports",
		Usage: "Deprecated flag, use --udp.ports to set range of listen ports",
		Value: "0:0",
	}

	// FlagConsumer sets to run as consumer only which allows to skip bootstrap for some of the dependencies.
	FlagConsumer = cli.BoolFlag{
		Name:  "consumer",
		Usage: "Run in consumer mode only.",
		Value: false,
	}

	// FlagDefaultCurrency sets the default currency used in node
	FlagDefaultCurrency = cli.StringFlag{
		Name:   metadata.FlagNames.DefaultCurrency,
		Usage:  "Default currency used in node and apps that depend on it",
		Value:  metadata.DefaultNetwork.DefaultCurrency,
		Hidden: true, // Users are not meant to touch or see this.
	}

	// FlagDocsURL sets the URL which leads to node documentation.
	FlagDocsURL = cli.StringFlag{
		Name:   "docs-url",
		Usage:  "URL leading to node documentation",
		Value:  "https://docs.mysterium.network",
		Hidden: true,
	}

	// FlagDNSResolutionHeadstart sets the dns resolution head start for swarm dialer.
	FlagDNSResolutionHeadstart = cli.DurationFlag{
		Name:   "dns-resolution-headstart",
		Usage:  "the headstart we give DNS lookups versus IP lookups",
		Value:  time.Millisecond * 1500,
		Hidden: true,
	}

	// FlagResidentCountry sets the resident country
	FlagResidentCountry = cli.StringFlag{
		Name:  "resident-country",
		Usage: "set resident country. If not set initially a default country will be resolved.",
	}

	// FlagWireguardMTU sets Wireguard myst interface MTU.
	FlagWireguardMTU = cli.IntFlag{
		Name:  "wireguard.mtu",
		Usage: "Wireguard interface MTU",
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
	RegisterFlagsAffiliator(flags)
	RegisterFlagsPayments(flags)
	RegisterFlagsPolicy(flags)
	RegisterFlagsMMN(flags)
	RegisterFlagsPilvytis(flags)
	RegisterFlagsChains(flags)
	RegisterFlagsUI(flags)
	RegisterFlagsBlockchainNetwork(flags)
	RegisterFlagsSSE(flags)

	*flags = append(*flags,
		&FlagBindAddress,
		&FlagDiscoveryType,
		&FlagDiscoveryPingInterval,
		&FlagDiscoveryFetchInterval,
		&FlagDHTAddress,
		&FlagDHTPort,
		&FlagDHTProtocol,
		&FlagDHTBootstrapPeers,
		&FlagFeedbackURL,
		&FlagFirewallKillSwitch,
		&FlagFirewallProtectedNetworks,
		&FlagShaperEnabled,
		&FlagShaperBandwidth,
		&FlagKeystoreLightweight,
		&FlagLogHTTP,
		&FlagLogLevel,
		&FlagVerbose,
		&FlagOpenvpnBinary,
		&FlagQualityType,
		&FlagQualityAddress,
		&FlagTequilapiAddress,
		&FlagTequilapiAllowedHostnames,
		&FlagTequilapiPort,
		&FlagTequilapiUsername,
		&FlagTequilapiPassword,
		&FlagPProfEnable,
		&FlagUserMode,
		&FlagDVPNMode,
		&FlagProxyMode,
		&FlagProvCheckerMode,
		&FlagUserspace,
		&FlagVendorID,
		&FlagLauncherVersion,
		&FlagP2PListenPorts,
		&FlagConsumer,
		&FlagDefaultCurrency,
		&FlagDocsURL,
		&FlagDNSResolutionHeadstart,
		&FlagResidentCountry,
		&FlagWireguardMTU,
	)

	return nil
}

// ParseFlagsNode function fills in node options from CLI context
func ParseFlagsNode(ctx *cli.Context) {
	ParseFlagsDirectory(ctx)

	ParseFlagsLocation(ctx)
	ParseFlagsNetwork(ctx)
	ParseFlagsTransactor(ctx)
	ParseFlagsAffiliator(ctx)
	ParseFlagsPayments(ctx)
	ParseFlagsPolicy(ctx)
	ParseFlagsMMN(ctx)
	ParseFlagPilvytis(ctx)
	ParseFlagsChains(ctx)
	ParseFlagsUI(ctx)
	ParseFlagsSSE(ctx)
	//it is important to have this one at the end so it overwrites defaults correctly
	ParseFlagsBlockchainNetwork(ctx)

	Current.ParseStringFlag(ctx, FlagBindAddress)
	Current.ParseStringSliceFlag(ctx, FlagDiscoveryType)
	Current.ParseDurationFlag(ctx, FlagDiscoveryPingInterval)
	Current.ParseDurationFlag(ctx, FlagDiscoveryFetchInterval)
	Current.ParseStringFlag(ctx, FlagDHTAddress)
	Current.ParseIntFlag(ctx, FlagDHTPort)
	Current.ParseStringFlag(ctx, FlagDHTProtocol)
	Current.ParseStringSliceFlag(ctx, FlagDHTBootstrapPeers)
	Current.ParseStringFlag(ctx, FlagFeedbackURL)
	Current.ParseBoolFlag(ctx, FlagFirewallKillSwitch)
	Current.ParseStringFlag(ctx, FlagFirewallProtectedNetworks)
	Current.ParseBoolFlag(ctx, FlagShaperEnabled)
	Current.ParseUInt64Flag(ctx, FlagShaperBandwidth)
	Current.ParseBoolFlag(ctx, FlagKeystoreLightweight)
	Current.ParseBoolFlag(ctx, FlagLogHTTP)
	Current.ParseBoolFlag(ctx, FlagVerbose)
	Current.ParseStringFlag(ctx, FlagLogLevel)
	Current.ParseStringFlag(ctx, FlagOpenvpnBinary)
	Current.ParseStringFlag(ctx, FlagQualityAddress)
	Current.ParseStringFlag(ctx, FlagQualityType)
	Current.ParseStringFlag(ctx, FlagTequilapiAddress)
	Current.ParseStringFlag(ctx, FlagTequilapiAllowedHostnames)
	Current.ParseIntFlag(ctx, FlagTequilapiPort)
	Current.ParseStringFlag(ctx, FlagTequilapiUsername)
	Current.ParseStringFlag(ctx, FlagTequilapiPassword)
	Current.ParseBoolFlag(ctx, FlagPProfEnable)
	Current.ParseBoolFlag(ctx, FlagUserMode)
	Current.ParseBoolFlag(ctx, FlagDVPNMode)
	Current.ParseBoolFlag(ctx, FlagProxyMode)
	Current.ParseBoolFlag(ctx, FlagProvCheckerMode)

	Current.ParseBoolFlag(ctx, FlagUserspace)
	Current.ParseStringFlag(ctx, FlagVendorID)
	Current.ParseStringFlag(ctx, FlagLauncherVersion)
	Current.ParseStringFlag(ctx, FlagP2PListenPorts)
	Current.ParseBoolFlag(ctx, FlagConsumer)
	Current.ParseStringFlag(ctx, FlagDefaultCurrency)
	Current.ParseStringFlag(ctx, FlagDocsURL)
	Current.ParseDurationFlag(ctx, FlagDNSResolutionHeadstart)
	Current.ParseIntFlag(ctx, FlagWireguardMTU)

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

// ValidateWireguardMTUFlag validates given mtu flag
func ValidateWireguardMTUFlag() error {

	v := Current.GetInt(FlagWireguardMTU.Name)
	if v == 0 {
		return nil
	}
	if v < 68 || v > 1500 {
		msg := "Wireguard MTU value is out of possible range: 68..1500"
		log.Error().Msg(msg)
		return errors.Errorf("Flag validation error: %s", msg)
	}
	return nil
}
