/*
 * Copyright (C) 2021 The "MysteriumNetwork/node" Authors.
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

package cli

import (
	"fmt"

	"github.com/mysteriumnetwork/node/cmd/commands/cli/clio"
	"github.com/mysteriumnetwork/node/datasize"
	"github.com/mysteriumnetwork/node/money"
	"github.com/mysteriumnetwork/node/tequilapi/contract"
)

func (c *cliApp) service(args []string) (err error) {
	if len(args) == 0 {
		fmt.Println(serviceHelp)
		return errWrongArgumentCount
	}

	action := args[0]
	switch action {
	case "start":
		if len(args) < 3 {
			fmt.Println(serviceHelp)
			return errWrongArgumentCount
		}
		return c.serviceStart(args[1], args[2], args[3:]...)
	case "stop":
		if len(args) < 2 {
			fmt.Println(serviceHelp)
			return errWrongArgumentCount
		}
		return c.serviceStop(args[1])
	case "status":
		if len(args) < 2 {
			fmt.Println(serviceHelp)
			return errWrongArgumentCount
		}
		return c.serviceGet(args[1])
	case "list":
		return c.serviceList()
	case "sessions":
		return c.serviceSessions()
	default:
		fmt.Println(serviceHelp)
		return errUnknownSubCommand(args[0])
	}
}

func (c *cliApp) serviceStart(providerID, serviceType string, args ...string) (err error) {
	serviceOpts, err := parseStartFlags(serviceType, args...)
	if err != nil {
		return fmt.Errorf("failed to parse service options: %w", err)
	}

	service, err := c.tequilapi.ServiceStart(contract.ServiceStartRequest{
		ProviderID:     providerID,
		Type:           serviceType,
		AccessPolicies: &contract.ServiceAccessPolicies{IDs: serviceOpts.AccessPolicyList},
		Options:        serviceOpts.TypeOptions,
	})
	if err != nil {
		return fmt.Errorf("failed to start service: %w", err)
	}

	clio.Status(service.Status,
		"ID: "+service.ID,
		"ProviderID: "+service.Proposal.ProviderID,
		"Type: "+service.Proposal.ServiceType)
	return nil
}

func (c *cliApp) serviceStop(id string) (err error) {
	if err := c.tequilapi.ServiceStop(id); err != nil {
		return fmt.Errorf("failed to stop service: %w", err)
	}

	clio.Status("Stopping", "ID: "+id)
	return nil
}

func (c *cliApp) serviceList() (err error) {
	services, err := c.tequilapi.Services()
	if err != nil {
		return fmt.Errorf("failed to get a list of services: %w", err)
	}

	for _, service := range services {
		clio.Status(service.Status,
			"ID: "+service.ID,
			"ProviderID: "+service.Proposal.ProviderID,
			"Type: "+service.Proposal.ServiceType)
	}
	return nil
}

func (c *cliApp) serviceSessions() (err error) {
	sessions, err := c.tequilapi.Sessions()
	if err != nil {
		return fmt.Errorf("failed to get a list of sessions: %w", err)
	}

	clio.Status("Current sessions", len(sessions.Items))
	for _, session := range sessions.Items {
		clio.Status(
			"ID: "+session.ID,
			"ConsumerID: "+session.ConsumerID,
			fmt.Sprintf("Data: %s/%s", datasize.FromBytes(session.BytesReceived).String(), datasize.FromBytes(session.BytesSent).String()),
			fmt.Sprintf("Tokens: %s", money.New(session.Tokens)),
		)
	}
	return nil
}

func (c *cliApp) serviceGet(id string) (err error) {
	service, err := c.tequilapi.Service(id)
	if err != nil {
		return fmt.Errorf("failed to get service info: %w", err)
	}

	clio.Status(service.Status,
		"ID: "+service.ID,
		"ProviderID: "+service.Proposal.ProviderID,
		"Type: "+service.Proposal.ServiceType)
	return nil
}
