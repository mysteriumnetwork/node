/*
 * Copyright (C) 2018 The "MysteriumNetwork/node" Authors.
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

package registry

import (
	"github.com/MysteriumNetwork/payments/registry"
	log "github.com/cihub/seelog"
)

// PrintRegistrationData prints identity registration data needed to register identity with payments contract
func PrintRegistrationData(data *registry.RegistrationData) {
	infoColor := "\033[93m"
	stopColor := "\033[0m"
	log.Info(infoColor)
	log.Info("Identity is not registered yet. In order to do that - please call payments contract with the following data")
	log.Infof("Public key: part1 -> 0x%X", data.PublicKey.Part1)
	log.Infof("            part2 -> 0x%X", data.PublicKey.Part2)
	log.Infof("Signature: S -> 0x%X", data.Signature.S)
	log.Infof("           R -> 0x%X", data.Signature.R)
	log.Infof("           V -> 0x%X", data.Signature.V)
	log.Info("OR")
	log.Info("Execute the following link: ")
	log.Infof("https://wallet.mysterium.network/?part1=0x%X&part2=0x%X&s=0x%X&r=0x%X&v=0x%X%v",
		data.PublicKey.Part1,
		data.PublicKey.Part2,
		data.Signature.S,
		data.Signature.R,
		data.Signature.V, stopColor)
}
