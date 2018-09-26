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

package noop

import discovery_dto "github.com/mysteriumnetwork/node/service_discovery/dto"

// PromiseProcessor process promises in such way, what no actual money is deducted from promise
type PromiseProcessor struct{}

// Start processing promises for given service proposal
func (processor *PromiseProcessor) Start(proposal discovery_dto.ServiceProposal) error {
	return nil
}

// Stop stops processing promises
func (processor *PromiseProcessor) Stop() error {
	return nil
}
