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

	"github.com/mysteriumnetwork/node/cmd"
	"github.com/mysteriumnetwork/node/cmd/commands/daemon"
	"github.com/mysteriumnetwork/node/cmd/commands/license"
	"github.com/mysteriumnetwork/node/core/service"
	"github.com/mysteriumnetwork/node/metadata"
	"github.com/mysteriumnetwork/node/utils"
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

	agreedTermsConditionsFlag = cli.BoolFlag{
		Name:  "agreed-terms-and-conditions",
		Usage: "Agree with terms & conditions",
	}
)

// NewCommand function creates service command
func NewCommand(licenseCommandName string) *cli.Command {
	var di cmd.Dependencies

	stopCommand := func() error {
		errorServiceManager := di.ServiceManager.Kill()
		di.Shutdown()

		return errorServiceManager
	}

	return &cli.Command{
		Name:      serviceCommandName,
		Usage:     "Starts and publishes service on Mysterium Network",
		ArgsUsage: " ",
		Flags: []cli.Flag{
			identityFlag, identityPassphraseFlag,
			openvpnProtocolFlag, openvpnPortFlag,
			agreedTermsConditionsFlag,
		},
		Action: func(ctx *cli.Context) error {
			if !ctx.Bool(agreedTermsConditionsFlag.Name) {
				printTermWarning(licenseCommandName)
				os.Exit(2)
			}

			errorChannel := make(chan error, 1)
			daemon.StartDaemon(ctx, &di, errorChannel)

			di.BootstrapServiceComponents(cmd.ParseFlagsNode(ctx), service.Options{
				ctx.String(identityFlag.Name),
				ctx.String(identityPassphraseFlag.Name),

				ctx.String(openvpnProtocolFlag.Name),
				ctx.Int(openvpnPortFlag.Name),
			})

			go func() {
				if err := di.ServiceManager.Start(); err != nil {
					errorChannel <- err
					return
				}
				errorChannel <- di.ServiceManager.Wait()
			}()

			cmd.RegisterSignalCallback(utils.SoftKiller(stopCommand))

			err := <-errorChannel
			switch err {
			case service.ErrorLocation:
				printLocationWarning("myst")
				return nil
			default:
				return err
			}
		},
	}
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
