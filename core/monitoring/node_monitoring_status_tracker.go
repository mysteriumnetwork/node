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

package monitoring

import (
	"fmt"
	"github.com/mysteriumnetwork/node/core/quality"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/rs/zerolog/log"
)

// MonitoringStatus enum
type MonitoringStatus string

const (
	// Success enum
	Success MonitoringStatus = "success"
	// Failed enum
	Failed MonitoringStatus = "failed"
	// Pending enum
	Pending MonitoringStatus = "pending"
	// Unknown enum
	Unknown MonitoringStatus = "unknown"
)

type currentIdentity interface {
	GetUnlockedIdentity() (identity.Identity, bool)
}

type monitoringStatusApi interface {
	MonitoringStatus(providerIds []string) quality.MonitoringStatusResponse
}

// Session represent session monitoring state
type Session struct {
	ProviderID       string
	ServiceType      string
	MonitoringFailed bool
}

// MonitoringStatusTracker tracks node status for service
type MonitoringStatusTracker struct {
	currentIdentity     currentIdentity
	monitoringStatusApi monitoringStatusApi
}

// NewMonitoringStatusTracker constructor
func NewMonitoringStatusTracker(
	currentIdentity currentIdentity,
	monitoringStatusApi monitoringStatusApi,
) *MonitoringStatusTracker {
	return &MonitoringStatusTracker{
		currentIdentity:     currentIdentity,
		monitoringStatusApi: monitoringStatusApi,
	}
}

// Status retrieves and resolved monitoring status from quality oracle
func (k *MonitoringStatusTracker) Status() MonitoringStatus {
	id, ok := k.currentIdentity.GetUnlockedIdentity()

	if !ok {
		return "unknown"
	}

	response := k.monitoringStatusApi.MonitoringStatus([]string{id.Address})
	value, ok := response[id.Address]
	if !ok {
		log.Error().Msg(fmt.Sprintf("Monitoring status information was not received for: %s", id.Address))
		return "unknown"
	}

	return MonitoringStatus(value.MonitoringStatus)
}
