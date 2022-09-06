/*
 * Copyright (C) 2022 The "MysteriumNetwork/node" Authors.
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

package node

import (
	"github.com/pkg/errors"

	"github.com/mysteriumnetwork/node/identity"
)

// MonitoringAgentStatuses a object represent a [service_type][status]amount of statuses for each service type.
type MonitoringAgentStatuses map[string]map[string]int

// ProviderStatuses should return provider statuses from monitoring agent
type ProviderStatuses func(providerID string) (MonitoringAgentStatuses, error)

// ProviderSessionsList should return provider sessions list
type ProviderSessionsList func(id identity.Identity, rangeTime string) ([]SessionItem, error)

// ProviderTransferredData should return total traffic served by the provider during a period of time
type ProviderTransferredData func(id identity.Identity, rangeTime string) (TransferredData, error)

// ProviderSessionsCount should return provider sessions count
type ProviderSessionsCount func(id identity.Identity, rangeTime string) (SessionsCount, error)

// ProviderConsumersCount should return unique consumers count
type ProviderConsumersCount func(id identity.Identity, rangeTime string) (ConsumersCount, error)

// ProviderSeriesEarnings should return earnings data series metrics
type ProviderSeriesEarnings func(id identity.Identity, rangeTime string) (SeriesEarnings, error)

// ProviderSeriesSessions should return sessions data series metrics
type ProviderSeriesSessions func(id identity.Identity, rangeTime string) (SeriesSessions, error)

// ProviderSeriesData should return transferred bytes data series metrics
type ProviderSeriesData func(id identity.Identity, rangeTime string) (SeriesData, error)

// StatsTracker tracks metrics for service
type StatsTracker struct {
	providerStatuses        ProviderStatuses
	providerSessionsList    ProviderSessionsList
	providerTransferredData ProviderTransferredData
	providerSessionsCount   ProviderSessionsCount
	providerConsumersCount  ProviderConsumersCount
	providerSeriesEarnings  ProviderSeriesEarnings
	providerSeriesSessions  ProviderSeriesSessions
	providerSeriesData      ProviderSeriesData
	currentIdentity         currentIdentity
}

// NewNodeStatsTracker constructor
func NewNodeStatsTracker(
	providerStatuses ProviderStatuses,
	providerSessions ProviderSessionsList,
	providerTransferredData ProviderTransferredData,
	providerSessionsCount ProviderSessionsCount,
	providerConsumersCount ProviderConsumersCount,
	providerSeriesEarnings ProviderSeriesEarnings,
	providerSeriesSessions ProviderSeriesSessions,
	providerSeriesData ProviderSeriesData,
	currentIdentity currentIdentity,
) *StatsTracker {
	mat := &StatsTracker{
		providerStatuses:        providerStatuses,
		providerSessionsList:    providerSessions,
		providerTransferredData: providerTransferredData,
		providerSessionsCount:   providerSessionsCount,
		providerConsumersCount:  providerConsumersCount,
		providerSeriesEarnings:  providerSeriesEarnings,
		providerSeriesSessions:  providerSeriesSessions,
		providerSeriesData:      providerSeriesData,
		currentIdentity:         currentIdentity,
	}

	return mat
}

// Statuses retrieves and resolved monitoring status from quality oracle
func (m *StatsTracker) Statuses() (MonitoringAgentStatuses, error) {
	id, ok := m.currentIdentity.GetUnlockedIdentity()
	if ok {
		return m.providerStatuses(id.Address)
	}

	return MonitoringAgentStatuses{}, errors.New("identity not found")
}

// SessionItem represents information about session monitoring metrics.
type SessionItem struct {
	ID              string `json:"id"`
	ConsumerCountry string `json:"consumer_country"`
	ServiceType     string `json:"service_type"`
	Duration        int64  `json:"duration"`
	StartedAt       int64  `json:"started_at"`
	Earning         string `json:"earning"`
	Transferred     int64  `json:"transferred"`
}

// TransferredData represent information about total traffic served by the provider during a period of time
type TransferredData struct {
	Bytes int `json:"transferred_data_bytes"`
}

// SessionsCount represent a information about number of sessions during a period of time
type SessionsCount struct {
	Count int `json:"count"`
}

// ConsumersCount represent a information about number of consumers served during a period of time
type ConsumersCount struct {
	Count int `json:"count"`
}

// SeriesEarnings represents data series metrics about earnings during a time
type SeriesEarnings struct {
	Data []SeriesEarningItem `json:"data"`
}

// SeriesEarningItem represents an item with earning in MYSTT and timestamp
type SeriesEarningItem struct {
	Earning   float64 `json:"value"`
	Timestamp int64   `json:"timestamp"`
}

// SeriesSessions represents data series metrics about started sessions during a time
type SeriesSessions struct {
	Data []SeriesSessionItem `json:"data"`
}

// SeriesSessionItem represents an item with timestamp session event start
type SeriesSessionItem struct {
	Count     int   `json:"value"`
	Timestamp int64 `json:"timestamp"`
}

// SeriesData represents data series metrics about transferred bytes during a time
type SeriesData struct {
	Data []SeriesDataItem `json:"data"`
}

// SeriesDataItem represents an item with bytes transferred and timestamp
type SeriesDataItem struct {
	Bytes     int   `json:"value"`
	Timestamp int64 `json:"timestamp"`
}

// Sessions retrieves and resolved monitoring status from quality oracle
func (m *StatsTracker) Sessions(rangeTime string) ([]SessionItem, error) {
	id, ok := m.currentIdentity.GetUnlockedIdentity()
	if ok {
		return m.providerSessionsList(id, rangeTime)
	}

	return []SessionItem{}, errors.New("identity not found")
}

// TransferredData retrieves and resolved total traffic served by the provider
func (m *StatsTracker) TransferredData(rangeTime string) (TransferredData, error) {
	id, ok := m.currentIdentity.GetUnlockedIdentity()
	if ok {
		return m.providerTransferredData(id, rangeTime)
	}

	return TransferredData{}, errors.New("identity not found")
}

// SessionsCount retrieves and resolved numbers of sessions
func (m *StatsTracker) SessionsCount(rangeTime string) (SessionsCount, error) {
	id, ok := m.currentIdentity.GetUnlockedIdentity()
	if ok {
		return m.providerSessionsCount(id, rangeTime)
	}

	return SessionsCount{}, errors.New("identity not found")
}

// ConsumersCount retrieves and resolved numbers of consumers server during period of time
func (m *StatsTracker) ConsumersCount(rangeTime string) (ConsumersCount, error) {
	id, ok := m.currentIdentity.GetUnlockedIdentity()
	if ok {
		return m.providerConsumersCount(id, rangeTime)
	}

	return ConsumersCount{}, errors.New("identity not found")
}

// SeriesEarnings retrieves and resolved earnings data series metrics during a time range
func (m *StatsTracker) SeriesEarnings(rangeTime string) (SeriesEarnings, error) {
	id, ok := m.currentIdentity.GetUnlockedIdentity()
	if ok {
		return m.providerSeriesEarnings(id, rangeTime)
	}

	return SeriesEarnings{}, errors.New("identity not found")
}

// SeriesSessions retrieves and resolved sessions data series metrics during a time range
func (m *StatsTracker) SeriesSessions(rangeTime string) (SeriesSessions, error) {
	id, ok := m.currentIdentity.GetUnlockedIdentity()
	if ok {
		return m.providerSeriesSessions(id, rangeTime)
	}

	return SeriesSessions{}, errors.New("identity not found")
}

// SeriesData retrieves and resolved transferred bytes data series metrics during a time range
func (m *StatsTracker) SeriesData(rangeTime string) (SeriesData, error) {
	id, ok := m.currentIdentity.GetUnlockedIdentity()
	if ok {
		return m.providerSeriesData(id, rangeTime)
	}

	return SeriesData{}, errors.New("identity not found")
}
