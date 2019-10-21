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
	"strings"

	"github.com/rs/zerolog/log"
)

type serviceIPForward struct {
	CommandEnable  []string
	CommandDisable []string
	CommandRead    []string
	CommandFactory CommandFactory
	forward        bool
}

// CommandFactory is responsible for creating new instances of command
type CommandFactory func(name string, arg ...string) Command

// Command allows us to run commands
type Command interface {
	CombinedOutput() ([]byte, error)
	Output() ([]byte, error)
}

func (service *serviceIPForward) Enable() error {
	if service.Enabled() {
		service.forward = true
		log.Info().Msg("IP forwarding already enabled")
		return nil
	}

	if output, err := service.CommandFactory(service.CommandEnable[0], service.CommandEnable[1:]...).CombinedOutput(); err != nil {
		log.Warn().Err(err).Msgf("Failed to enable IP forwarding: %v Cmd output: %v", service.CommandEnable[1:], string(output))
		return err
	}

	log.Info().Msg("IP forwarding enabled")
	return nil
}

func (service *serviceIPForward) Disable() {
	if service.forward {
		return
	}

	if output, err := service.CommandFactory(service.CommandDisable[0], service.CommandDisable[1:]...).CombinedOutput(); err != nil {
		log.Warn().Err(err).Msgf("Failed to disable IP forwarding: %v Cmd output: %v", service.CommandDisable[1:], string(output))
	}

	log.Info().Msg("IP forwarding disabled")
}

func (service *serviceIPForward) Enabled() bool {
	output, err := service.CommandFactory(service.CommandRead[0], service.CommandRead[1:]...).Output()
	if err != nil {
		log.Warn().Err(err).Msgf("Failed to check IP forwarding status: %v Cmd output: %v", service.CommandRead[1:], string(output))
	}

	return strings.TrimSpace(string(output)) == "1"
}
