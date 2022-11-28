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

	"github.com/urfave/cli/v2"

	"github.com/mysteriumnetwork/node/metadata"
)

var (
	// FlagBlockchainNetwork uses specified blockchain network.
	FlagBlockchainNetwork = cli.StringFlag{
		Name:  "network",
		Usage: "Defines default blockchain network configuration",
		Value: string(Mainnet),
	}
	// FlagAPIAddress Mysterium API URL
	// Deprecated: use FlagDiscoveryAddress
	FlagAPIAddress = cli.StringFlag{
		Name:  metadata.FlagNames.MysteriumAPIAddress,
		Usage: "Deprecated flag. Use `discovery.address` flag instead to specify URL of Discovery API",
		Value: metadata.DefaultNetwork.MysteriumAPIAddress,
	}
	// FlagDiscoveryAddress discovery url
	FlagDiscoveryAddress = cli.StringFlag{
		Name:  metadata.FlagNames.DiscoveryAddress,
		Usage: "URL of Discovery API",
		Value: metadata.DefaultNetwork.DiscoveryAddress,
	}
	// FlagChainID chain id to use
	FlagChainID = cli.Int64Flag{
		Name:  metadata.FlagNames.DefaultChainIDFlag,
		Usage: "The chain ID to use",
		Value: metadata.DefaultNetwork.DefaultChainID,
	}
	// FlagBrokerAddress message broker URI.
	FlagBrokerAddress = cli.StringSliceFlag{
		Name:  metadata.FlagNames.BrokerAddressesFlag,
		Usage: "URI of message broker",
		Value: cli.NewStringSlice(metadata.DefaultNetwork.BrokerAddresses...),
	}
	// FlagEtherRPCL1 URL or IPC socket to connect to Ethereum node.
	FlagEtherRPCL1 = cli.StringSliceFlag{
		Name:  metadata.FlagNames.Chain1Flag.EtherClientRPCFlag,
		Usage: "L1 URL or IPC socket to connect to ethereum node, anything what ethereum client accepts - works",
		Value: cli.NewStringSlice(metadata.DefaultNetwork.Chain1.EtherClientRPC...),
	}
	// FlagEtherRPCL2 URL or IPC socket to connect to Ethereum node.
	FlagEtherRPCL2 = cli.StringSliceFlag{
		Name:  metadata.FlagNames.Chain2Flag.EtherClientRPCFlag,
		Usage: "L2 URL or IPC socket to connect to ethereum node, anything what ethereum client accepts - works",
		Value: cli.NewStringSlice(metadata.DefaultNetwork.Chain2.EtherClientRPC...),
	}
	// FlagNATHolePunching remove the deprecated flag once all users stop to call it.
	FlagNATHolePunching = cli.BoolFlag{
		Name:    "nat-hole-punching",
		Aliases: []string{"experiment-natpunching"}, // TODO: remove the deprecated alias once all users stop to use it.
		Usage:   "Deprecated flag use `traversal` flag instead to disable or enable methods",
		Value:   true,
	}
	// FlagPortMapping enables NAT port mapping.
	FlagPortMapping = cli.BoolFlag{
		Name:  "nat-port-mapping",
		Usage: "Deprecated flag use `traversal` flag instead to disable or enable methods",
		Value: true,
	}
	// FlagIncomingFirewall enables incoming traffic filtering.
	FlagIncomingFirewall = cli.BoolFlag{
		Name:  "incoming-firewall",
		Usage: "Enables incoming traffic filtering",
		Value: false,
	}
	// FlagOutgoingFirewall enables outgoing traffic filtering.
	FlagOutgoingFirewall = cli.BoolFlag{
		Name:  "outgoing-firewall",
		Usage: "Enables outgoing traffic filtering",
		Value: false,
	}
	// FlagKeepConnectedOnFail keeps connection active to prevent traffic leaks.
	FlagKeepConnectedOnFail = cli.BoolFlag{
		Name:  "keep-connected-on-fail",
		Usage: "Do not disconnect consumer on session fail to prevent traffic leaks",
		Value: false,
	}
	// FlagAutoReconnect restore connection automatically once it failed.
	FlagAutoReconnect = cli.BoolFlag{
		Name:  "auto-reconnect",
		Usage: "Restore connection automatically once it failed",
		Value: false,
	}
	// FlagSTUNservers list of STUN server to be used to detect NAT type.
	FlagSTUNservers = cli.StringSliceFlag{
		Name:  "stun-servers",
		Usage: "Comma separated list of STUN server to be used to detect NAT type",
		Value: cli.NewStringSlice("stun.l.google.com:19302", "stun1.l.google.com:19302", "stun2.l.google.com:19302"),
	}
	// FlagLocalServiceDiscovery enables SSDP and Bonjour local service discovery.
	FlagLocalServiceDiscovery = cli.BoolFlag{
		Name:  "local-service-discovery",
		Usage: "Enables SSDP and Bonjour local service discovery",
		Value: true,
	}
	// FlagUDPListenPorts sets allowed UDP port range for listening.
	FlagUDPListenPorts = cli.StringFlag{
		Name:  "udp.ports",
		Usage: "Range of UDP listen ports used for connections",
		Value: "10000:60000",
	}
	// FlagTraversal order of NAT traversal methods to be used for providing service.
	FlagTraversal = cli.StringFlag{
		Name:  "traversal",
		Usage: "Comma separated order of NAT traversal methods to be used for providing service",
		Value: "manual,upnp,holepunching",
	}
	// FlagPortCheckServers list of asymmetric UDP echo servers for checking port availability
	FlagPortCheckServers = cli.StringFlag{
		Name:   "port-check-servers",
		Usage:  "Comma separated list of asymmetric UDP echo servers for checking port availability",
		Value:  "echo.mysterium.network:4589",
		Hidden: true,
	}

	// FlagStatsReportInterval is interval for consumer connection statistics reporting.
	FlagStatsReportInterval = cli.DurationFlag{
		Name:   "stats-report-interval",
		Usage:  "Duration between syncing stats from the network interface with a node",
		Value:  1 * time.Second,
		Hidden: true,
	}

	// FlagDNSListenPort sets the port for listening by DNS service.
	FlagDNSListenPort = cli.IntFlag{
		Name:  "dns.listen-port",
		Usage: "DNS listen port for services",
		Value: 11253,
	}
)

// RegisterFlagsNetwork function register network flags to flag list
func RegisterFlagsNetwork(flags *[]cli.Flag) {
	*flags = append(
		*flags,
		&FlagPortMapping,
		&FlagNATHolePunching,
		&FlagAPIAddress,
		&FlagDiscoveryAddress,
		&FlagBrokerAddress,
		&FlagEtherRPCL1,
		&FlagEtherRPCL2,
		&FlagIncomingFirewall,
		&FlagOutgoingFirewall,
		&FlagChainID,
		&FlagKeepConnectedOnFail,
		&FlagAutoReconnect,
		&FlagSTUNservers,
		&FlagLocalServiceDiscovery,
		&FlagUDPListenPorts,
		&FlagTraversal,
		&FlagPortCheckServers,
		&FlagStatsReportInterval,
		&FlagDNSListenPort,
	)
}

// ParseFlagsNetwork function fills in directory options from CLI context
func ParseFlagsNetwork(ctx *cli.Context) {
	Current.ParseStringFlag(ctx, FlagAPIAddress)
	Current.ParseStringFlag(ctx, FlagDiscoveryAddress)
	Current.ParseStringSliceFlag(ctx, FlagBrokerAddress)
	Current.ParseStringSliceFlag(ctx, FlagEtherRPCL1)
	Current.ParseStringSliceFlag(ctx, FlagEtherRPCL2)
	Current.ParseBoolFlag(ctx, FlagPortMapping)
	Current.ParseBoolFlag(ctx, FlagNATHolePunching)
	Current.ParseBoolFlag(ctx, FlagIncomingFirewall)
	Current.ParseBoolFlag(ctx, FlagOutgoingFirewall)
	Current.ParseInt64Flag(ctx, FlagChainID)
	Current.ParseBoolFlag(ctx, FlagKeepConnectedOnFail)
	Current.ParseBoolFlag(ctx, FlagAutoReconnect)
	Current.ParseStringSliceFlag(ctx, FlagSTUNservers)
	Current.ParseBoolFlag(ctx, FlagLocalServiceDiscovery)
	Current.ParseStringFlag(ctx, FlagUDPListenPorts)
	Current.ParseStringFlag(ctx, FlagTraversal)
	Current.ParseStringFlag(ctx, FlagPortCheckServers)
	Current.ParseDurationFlag(ctx, FlagStatsReportInterval)
	Current.ParseIntFlag(ctx, FlagDNSListenPort)
}

// BlockchainNetwork defines a blockchain network
type BlockchainNetwork string

var (
	// Mainnet defines the mainnet blockchain network
	Mainnet BlockchainNetwork = "mainnet"
	// Testnet defines the testnet blockchain network
	Testnet BlockchainNetwork = "testnet"
	// Localnet defines the localnet blockchain network
	Localnet BlockchainNetwork = "localnet"
)

// ParseBlockchainNetwork parses a string argument into blockchain network
func ParseBlockchainNetwork(network string) (BlockchainNetwork, error) {
	if isValidBlockchainNetwork(network) {
		return BlockchainNetwork(strings.ToLower(network)), nil
	}
	return Mainnet, fmt.Errorf("unknown blockchain network: %s", network)
}

func isValidBlockchainNetwork(network string) bool {
	parsedNetwork := BlockchainNetwork(strings.ToLower(network))
	return parsedNetwork.IsMainnet() || parsedNetwork.IsTestnet() || parsedNetwork.IsLocalnet()
}

// IsMainnet returns whether the blockchain network is mainnet or not
func (n BlockchainNetwork) IsMainnet() bool {
	return n == Mainnet
}

// IsTestnet returns whether the blockchain network is testnet or not
func (n BlockchainNetwork) IsTestnet() bool {
	return n == Testnet
}

// IsLocalnet returns whether the blockchain network is localnet or not
func (n BlockchainNetwork) IsLocalnet() bool {
	return n == Localnet
}

// ParseFlagsBlockchainNetwork function fills in directory options from CLI context
func ParseFlagsBlockchainNetwork(ctx *cli.Context) {
	Current.ParseBlockchainNetworkFlag(ctx, FlagBlockchainNetwork)
}

// RegisterFlagsBlockchainNetwork function registers blockchain network flags to flag list
func RegisterFlagsBlockchainNetwork(flags *[]cli.Flag) {
	*flags = append(
		*flags,
		&FlagBlockchainNetwork,
	)
}
