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

package cmd

import (
	"github.com/mysteriumnetwork/node/core/node"
	"github.com/mysteriumnetwork/node/logconfig"
	openvpn_core "github.com/mysteriumnetwork/node/services/openvpn/core"
	"github.com/urfave/cli"
)

var (
	tequilapiAddressFlag = cli.StringFlag{
		Name:  "tequilapi.address",
		Usage: "IP address of interface to listen for incoming connections",
		Value: "127.0.0.1",
	}
	tequilapiPortFlag = cli.IntFlag{
		Name:  "tequilapi.port",
		Usage: "Port for listening incoming api requests",
		Value: 4050,
	}
	keystoreLightweightFlag = cli.BoolFlag{
		Name:  "keystore.lightweight",
		Usage: "Determines the scrypt memory complexity. If set to true, will use 4MB blocks instead of the standard 256MB ones",
	}
)

// ParseKeystoreFlags parses the keystore options for node
func ParseKeystoreFlags(ctx *cli.Context) node.OptionsKeystore {
	return node.OptionsKeystore{
		UseLightweight: ctx.GlobalBool(keystoreLightweightFlag.Name),
	}
}

// RegisterFlagsNode function register node flags to flag list
func RegisterFlagsNode(flags *[]cli.Flag) error {
	if err := RegisterFlagsDirectory(flags); err != nil {
		return err
	}

	*flags = append(*flags, tequilapiAddressFlag, tequilapiPortFlag, keystoreLightweightFlag)

	RegisterFlagsNetwork(flags)
	RegisterFlagsDiscovery(flags)
	RegisterFlagsQuality(flags)
	openvpn_core.RegisterFlags(flags)
	RegisterFlagsLocation(flags)
	RegisterFlagsUI(flags)

	return nil
}

// ParseFlagsNode function fills in node options from CLI context
func ParseFlagsNode(ctx *cli.Context) node.Options {
	logconfig.ParseFlags(ctx)
	return node.Options{
		Directories: ParseFlagsDirectory(ctx),

		TequilapiAddress: ctx.GlobalString(tequilapiAddressFlag.Name),
		TequilapiPort:    ctx.GlobalInt(tequilapiPortFlag.Name),
		UI:               ParseFlagsUI(ctx),

		Keystore: ParseKeystoreFlags(ctx),

		OptionsNetwork: ParseFlagsNetwork(ctx),
		Discovery:      ParseFlagsDiscovery(ctx),
		Quality:        ParseFlagsQuality(ctx),
		Location:       ParseFlagsLocation(ctx),

		Openvpn: wrapper{nodeOptions: openvpn_core.ParseFlags(ctx)},
	}
}

// TODO this struct will disappear when we unify go-openvpn embedded lib and external process based session creation/handling
type wrapper struct {
	nodeOptions openvpn_core.NodeOptions
}

func (w wrapper) Check() error {
	return w.nodeOptions.Check()
}

func (w wrapper) BinaryPath() string {
	return w.nodeOptions.BinaryPath
}

var _ node.Openvpn = wrapper{}
