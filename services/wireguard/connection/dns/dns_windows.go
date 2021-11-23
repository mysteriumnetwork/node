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
	"fmt"
	"os/exec"
)

func setDNS(cfg Config) error {
	cmd := fmt.Sprintf("netsh interface ipv4 set dnsservers name=%s source=static address=%s validate=no", cfg.IfaceName, cfg.DNS[0])
	out, err := exec.Command("powershell", "-Command", cmd).CombinedOutput()
	if err != nil {
		return fmt.Errorf("could not configure DNS, %s:%v", string(out), err)
	}
	return nil
}

func cleanDNS(cfg Config) error {
	cmd := fmt.Sprintf("netsh interface ipv4 set dnsservers name=%s source=static address=none validate=no register=both", cfg.IfaceName)
	out, err := exec.Command("powershell", "-Command", cmd).CombinedOutput()
	if err != nil {
		return fmt.Errorf("could not clean DNS, %s:%w", string(out), err)
	}
	return nil
}
