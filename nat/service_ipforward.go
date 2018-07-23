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

package nat

import (
	log "github.com/cihub/seelog"
	"github.com/mysterium/node/utils"
	"os/exec"
	"strings"
)

type serviceIPForward struct {
	command  string
	variable string
	forward  bool
}

func (service *serviceIPForward) Enable() error {
	if service.Enabled() {
		service.forward = true
		log.Info(natLogPrefix, "IP forwarding already enabled")
		return nil
	}

	cmd := utils.SplitCommand(service.command, "-w net.inet.ip.forwarding=1")
	if output, err := cmd.CombinedOutput(); err != nil {
		log.Warn("Failed to enable IP forwarding: ", cmd.Args, " Returned exit error: ", err.Error(), " Cmd output: ", string(output))
		return err
	}

	log.Info(natLogPrefix, "IP forwarding enabled")
	return nil
}

func (service *serviceIPForward) Enabled() bool {
	cmd := exec.Command(service.command, "-n", service.variable)
	output, err := cmd.Output()
	if err != nil {
		log.Warn("Failed to check IP forwarding status: ", cmd.Args, " Returned exit error: ", err.Error(), " Cmd output: ", string(output))
	}

	if strings.TrimSpace(string(output)) == "1" {
		return true
	}
	return false
}

func (service *serviceIPForward) Disable() {
	if service.forward {
		return
	}

	cmd := utils.SplitCommand(service.command, "-w net.inet.ip.forwarding=0")
	if output, err := cmd.CombinedOutput(); err != nil {
		log.Warn("Failed to disable IP forwarding: ", cmd.Args, " Returned exit error: ", err.Error(), " Cmd output: ", string(output))
	} else {
		log.Info(natLogPrefix, "IP forwarding disabled")
	}
}
