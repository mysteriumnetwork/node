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
	"os/exec"
	"strings"

	log "github.com/cihub/seelog"
)

type serviceIPForward struct {
	CommandEnable  []string
	CommandDisable []string
	CommandRead    []string
	forward        bool
}

func (service *serviceIPForward) Enable() error {
	if service.Enabled() {
		service.forward = true
		log.Info(natLogPrefix, "IP forwarding already enabled")
		return nil
	}

	if output, err := exec.Command(service.CommandEnable[0], service.CommandEnable[1:]...).CombinedOutput(); err != nil {
		log.Warn("Failed to enable IP forwarding: ", service.CommandEnable[1:], " Returned exit error: ", err.Error(), " Cmd output: ", string(output))
		return err
	}

	log.Info(natLogPrefix, "IP forwarding enabled")
	return nil
}

func (service *serviceIPForward) Disable() {
	if service.forward {
		return
	}

	if output, err := exec.Command(service.CommandDisable[0], service.CommandDisable[1:]...).CombinedOutput(); err != nil {
		log.Warn("Failed to disable IP forwarding: ", service.CommandDisable[1:], " Returned exit error: ", err.Error(), " Cmd output: ", string(output))
	}

	log.Info(natLogPrefix, "IP forwarding disabled")
}

func (service *serviceIPForward) Enabled() bool {
	output, err := exec.Command(service.CommandRead[0], service.CommandRead[1:]...).Output()
	if err != nil {
		log.Warn("Failed to check IP forwarding status: ", service.CommandRead[1:], " Returned exit error: ", err.Error(), " Cmd output: ", string(output))
	}

	return strings.TrimSpace(string(output)) == "1"
}
