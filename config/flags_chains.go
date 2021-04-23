/*
 * Copyright (C) 2021 The "MysteriumNetwork/node" Authors.
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

	"github.com/mysteriumnetwork/node/metadata"
	"github.com/urfave/cli/v2"
)

// TODO: open to suggestions how to do this better.
var (
	// FlagChain1RegistryAddress represents the registry address for chain1.
	FlagChain1RegistryAddress = getRegistryFlag(1)
	// FlagChain2RegistryAddress represents the registry address for chain2.
	FlagChain2RegistryAddress = getRegistryFlag(2)
	// FlagChain1HermesAddress represents the hermes address for chain1.
	FlagChain1HermesAddress = getHermesIDFlag(1)
	// FlagChain2HermesAddress represents the hermes address for chain2.
	FlagChain2HermesAddress = getHermesIDFlag(2)
	// FlagChain1ChannelImplementationAddress represents the channel implementation address for chain1.
	FlagChain1ChannelImplementationAddress = getChannelImplementationFlag(1)
	// FlagChain2ChannelImplementationAddress represents the channel implementation address for chain2.
	FlagChain2ChannelImplementationAddress = getChannelImplementationFlag(2)
	// FlagChain1MystAddress represents the myst address for chain1.
	FlagChain1MystAddress = getMystAddressFlag(1)
	// FlagChain2MystAddress represents the myst address for chain2.
	FlagChain2MystAddress = getMystAddressFlag(2)
	// FlagChain1ChainID represents the chainID for chain1.
	FlagChain1ChainID = getChainIDFlag(1)
	// FlagChain1ChainID represents the chainID for chain2.
	FlagChain2ChainID = getChainIDFlag(2)
)

// RegisterFlagsChains function registers chain flags to flag list.
func RegisterFlagsChains(flags *[]cli.Flag) {
	*flags = append(*flags,
		&FlagChain1RegistryAddress,
		&FlagChain2RegistryAddress,
		&FlagChain1HermesAddress,
		&FlagChain2HermesAddress,
		&FlagChain1ChannelImplementationAddress,
		&FlagChain2ChannelImplementationAddress,
		&FlagChain1MystAddress,
		&FlagChain2MystAddress,
		&FlagChain1ChainID,
		&FlagChain2ChainID,
	)
}

// ParseFlagsChains function fills in chain options from CLI context.
func ParseFlagsChains(ctx *cli.Context) {
	Current.ParseStringFlag(ctx, FlagChain1RegistryAddress)
	Current.ParseStringFlag(ctx, FlagChain2RegistryAddress)
	Current.ParseStringFlag(ctx, FlagChain1HermesAddress)
	Current.ParseStringFlag(ctx, FlagChain2HermesAddress)
	Current.ParseStringFlag(ctx, FlagChain1ChannelImplementationAddress)
	Current.ParseStringFlag(ctx, FlagChain2ChannelImplementationAddress)
	Current.ParseStringFlag(ctx, FlagChain1MystAddress)
	Current.ParseStringFlag(ctx, FlagChain2MystAddress)
	Current.ParseInt64Flag(ctx, FlagChain1ChainID)
	Current.ParseInt64Flag(ctx, FlagChain2ChainID)
}

func getRegistryFlag(chainIndex int64) cli.StringFlag {
	defaultAddress := metadata.DefaultNetwork.Chain1.RegistryAddress
	if chainIndex == 2 {
		defaultAddress = metadata.DefaultNetwork.Chain2.RegistryAddress
	}

	return cli.StringFlag{
		Name:  fmt.Sprintf("chains.%v.registry", chainIndex),
		Value: defaultAddress,
		Usage: fmt.Sprintf("Sets the registry smart contract address for main chain %v", chainIndex),
	}
}

func getHermesIDFlag(chainIndex int64) cli.StringFlag {
	defaultAddress := metadata.DefaultNetwork.Chain1.HermesID
	if chainIndex == 2 {
		defaultAddress = metadata.DefaultNetwork.Chain2.HermesID
	}

	return cli.StringFlag{
		Name:  fmt.Sprintf("chains.%v.hermes", chainIndex),
		Value: defaultAddress,
		Usage: fmt.Sprintf("Sets the hermes smart contract address for chain %v", chainIndex),
	}
}

func getChannelImplementationFlag(chainIndex int64) cli.StringFlag {
	defaultAddress := metadata.DefaultNetwork.Chain1.ChannelImplAddress
	if chainIndex == 2 {
		defaultAddress = metadata.DefaultNetwork.Chain2.ChannelImplAddress
	}

	return cli.StringFlag{
		Name:  fmt.Sprintf("chains.%v.channelImplementation", chainIndex),
		Value: defaultAddress,
		Usage: fmt.Sprintf("Sets the channel implementation smart contract address for chain %v", chainIndex),
	}
}

func getMystAddressFlag(chainIndex int64) cli.StringFlag {
	defaultAddress := metadata.DefaultNetwork.Chain1.MystAddress
	if chainIndex == 2 {
		defaultAddress = metadata.DefaultNetwork.Chain2.MystAddress
	}

	return cli.StringFlag{
		Name:  fmt.Sprintf("chains.%v.myst", chainIndex),
		Value: defaultAddress,
		Usage: fmt.Sprintf("Sets the myst smart contract address for chain %v", chainIndex),
	}
}

func getChainIDFlag(chainIndex int64) cli.Int64Flag {
	defaultAddress := metadata.DefaultNetwork.Chain1.ChainID
	if chainIndex == 2 {
		defaultAddress = metadata.DefaultNetwork.Chain2.ChainID
	}

	return cli.Int64Flag{
		Name:  fmt.Sprintf("chains.%v.chainID", chainIndex),
		Value: defaultAddress,
		Usage: fmt.Sprintf("Sets the chainID for chain %v", chainIndex),
	}
}
