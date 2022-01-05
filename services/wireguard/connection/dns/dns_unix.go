//go:build !windows

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

package dns

import (
	"os"
	"os/exec"
	"path"
)

func setDNS(cfg Config) error {
	cmd := exec.Command(path.Join(cfg.ScriptDir, "update-resolv-conf"))
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, "script_type=up", "dev="+cfg.IfaceName, "foreign_option_1=dhcp-option DNS "+cfg.DNS[0])
	return cmd.Run()
}

func cleanDNS(cfg Config) error {
	cmd := exec.Command(path.Join(cfg.ScriptDir, "update-resolv-conf"))
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, "script_type=down", "dev="+cfg.IfaceName)
	return cmd.Run()
}
