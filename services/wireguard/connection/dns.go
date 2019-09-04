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

package connection

import (
	"os"
	"os/exec"
	"path"
)

func setDNS(configDir, dev, dns string) error {
	cmd := exec.Command(path.Join(configDir, "update-resolv-conf"))
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, "script_type=up", "dev="+dev, "foreign_option_1=dhcp-option DNS "+dns)
	return cmd.Run()
}

func cleanDNS(configDir, dev string) error {
	cmd := exec.Command(path.Join(configDir, "update-resolv-conf"))
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, "script_type=down", "dev="+dev)
	return cmd.Run()
}
