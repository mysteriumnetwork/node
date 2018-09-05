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
	"github.com/mysteriumnetwork/node/cmd"
	"github.com/mysteriumnetwork/node/core/service"
	"github.com/mysteriumnetwork/node/utils"
	"github.com/urfave/cli"
)

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

	openvpnProtocolFlag = cli.StringFlag{
		Name:  "openvpn.proto",
		Usage: "Openvpn protocol to use. Options: { udp, tcp }",
		Value: "udp",
	}
	openvpnPortFlag = cli.IntFlag{
		Name:  "openvpn.port",
		Usage: "Openvpn port to use. Default 1194",
		Value: 1194,
	}
)

// NewCommand function creates service command
func NewCommand() *cli.Command {
	var di cmd.Dependencies

	stopCommand := func() error {
		errorServiceManager := di.ServiceManager.Kill()
		errorNode := di.Node.Kill()

		if errorServiceManager != nil {
			return errorServiceManager
		}
		return errorNode
	}

	return &cli.Command{
		Name:      "service",
		Usage:     "Starts and publishes service on Mysterium Network",
		ArgsUsage: " ",
		Flags: []cli.Flag{
			identityFlag, identityPassphraseFlag,
			openvpnProtocolFlag, openvpnPortFlag,
		},
		Action: func(ctx *cli.Context) error {
			nodeOptions := cmd.ParseNodeFlags(ctx)
			di.Bootstrap(nodeOptions)
			di.BootstrapServiceManager(nodeOptions, service.Options{
				ctx.String(identityFlag.Name),
				ctx.String(identityPassphraseFlag.Name),

				ctx.String(openvpnProtocolFlag.Name),
				ctx.Int(openvpnPortFlag.Name),
			})

			errorChannel := make(chan error, 1)

			go func() {
				if err := di.Node.Start(); err != nil {
					errorChannel <- err
					return
				}
				errorChannel <- di.Node.Wait()
			}()

			go func() {
				if err := di.ServiceManager.Start(); err != nil {
					errorChannel <- err
					return
				}
				errorChannel <- di.ServiceManager.Wait()
			}()

			cmd.RegisterSignalCallback(utils.SoftKiller(stopCommand))

			return <-errorChannel
		},
		After: func(ctx *cli.Context) error {
			return stopCommand()
		},
	}
}
