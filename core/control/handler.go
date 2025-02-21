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

package control

import (
	"github.com/mysteriumnetwork/node/services"
	"github.com/mysteriumnetwork/node/tequilapi/contract"
	"github.com/rs/zerolog/log"
)

// handler is a function that handles control messages
func (c *ControlPlane) handler(request controlMessage) error {
	currentServices, err := c.api.Services()
	if err != nil {
		return err
	}

	for _, r := range request {
		if r.Command == "start" {
			log.Info().Str("service", r.Service).Msg("executing control start request")
			if err := c.startService(r.Service, c.identity); err != nil {
				log.Warn().AnErr("err", err).Msg("failed to start service")
			}
			continue
		}

		for _, service := range currentServices {
			if service.Type != r.Service {
				continue
			}

			if r.Command == "stop" {
				log.Info().Str("service", service.Type).Msg("executing control stop request")
				if err := c.stopService(service.ID); err != nil {
					log.Warn().AnErr("err", err).Msg("failed to start service")
				}
			}

			if r.Command == "restart" {
				log.Info().Str("service", service.Type).Msg("executing control restart request")
				if err := c.stopService(service.ID); err != nil {
					log.Warn().AnErr("err", err).Msg("failed to stop service")
				}
				if err := c.startService(service.Type, c.identity); err != nil {
					log.Warn().AnErr("err", err).Msg("failed to start service")
				}
			}
		}
	}
	return nil
}

func (c *ControlPlane) startService(serviceType, providerID string) error {
	serviceOpts, err := services.GetStartOptions(serviceType)
	if err != nil {
		return err
	}
	startRequest := contract.ServiceStartRequest{
		ProviderID:     providerID,
		Type:           serviceType,
		AccessPolicies: &contract.ServiceAccessPolicies{IDs: serviceOpts.AccessPolicyList},
		Options:        serviceOpts,
	}
	_, err = c.api.ServiceStart(startRequest)
	return err
}

func (c *ControlPlane) stopService(id string) error {
	return c.api.ServiceStop(id)
}
