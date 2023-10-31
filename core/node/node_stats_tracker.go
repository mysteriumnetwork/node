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

var errIdentityNotFound = errors.New("identity not found")

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

// ProviderEarningsSeries should return earnings data series metrics
type ProviderEarningsSeries func(id identity.Identity, rangeTime string) (EarningsSeries, error)

// ProviderSessionsSeries should return sessions data series metrics
type ProviderSessionsSeries func(id identity.Identity, rangeTime string) (SessionsSeries, error)

// ProviderTransferredDataSeries should return transferred bytes data series metrics
type ProviderTransferredDataSeries func(id identity.Identity, rangeTime string) (TransferredDataSeries, error)

// ProviderServiceEarnings should return earnings by service type
type ProviderServiceEarnings func(id identity.Identity) (EarningsPerService, error)

// ProviderActivityStats should return provider activity stats
type ProviderActivityStats func(id identity.Identity) (ActivityStats, error)

// ProviderQuality should return provider quality
type ProviderQuality func(id identity.Identity) (QualityInfo, error)

type currentIdentity interface {
	GetUnlockedIdentity() (identity.Identity, bool)
}

// StatsTracker tracks metrics for service
type StatsTracker struct {
	providerStatuses              ProviderStatuses
	providerSessionsList          ProviderSessionsList
	providerTransferredData       ProviderTransferredData
	providerSessionsCount         ProviderSessionsCount
	providerConsumersCount        ProviderConsumersCount
	providerEarningsSeries        ProviderEarningsSeries
	providerSessionsSeries        ProviderSessionsSeries
	providerTransferredDataSeries ProviderTransferredDataSeries
	providerActivityStats         ProviderActivityStats
	providerQuality               ProviderQuality
	providerServiceEarnings       ProviderServiceEarnings
	currentIdentity               currentIdentity
}

// NewNodeStatsTracker constructor
func NewNodeStatsTracker(
	providerStatuses ProviderStatuses,
	providerSessions ProviderSessionsList,
	providerTransferredData ProviderTransferredData,
	providerSessionsCount ProviderSessionsCount,
	providerConsumersCount ProviderConsumersCount,
	providerEarningsSeries ProviderEarningsSeries,
	providerSessionsSeries ProviderSessionsSeries,
	providerTransferredDataSeries ProviderTransferredDataSeries,
	providerActivityStats ProviderActivityStats,
	providerQuality ProviderQuality,
	providerServiceEarnings ProviderServiceEarnings,
	currentIdentity currentIdentity,
) *StatsTracker {
	mat := &StatsTracker{
		providerStatuses:              providerStatuses,
		providerSessionsList:          providerSessions,
		providerTransferredData:       providerTransferredData,
		providerSessionsCount:         providerSessionsCount,
		providerConsumersCount:        providerConsumersCount,
		providerEarningsSeries:        providerEarningsSeries,
		providerSessionsSeries:        providerSessionsSeries,
		providerTransferredDataSeries: providerTransferredDataSeries,
		providerActivityStats:         providerActivityStats,
		providerQuality:               providerQuality,
		providerServiceEarnings:       providerServiceEarnings,
		currentIdentity:               currentIdentity,
	}

	return mat
}

// Statuses retrieves and resolved monitoring status from quality oracle
func (m *StatsTracker) Statuses() (MonitoringAgentStatuses, error) {
	id, ok := m.currentIdentity.GetUnlockedIdentity()
	if ok {
		return m.providerStatuses(id.Address)
	}

	return MonitoringAgentStatuses{}, errIdentityNotFound
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

// SeriesItem represents a general data series item
type SeriesItem struct {
	Value     string `json:"value"`
	Timestamp int64  `json:"timestamp"`
}

// EarningsSeries represents data series metrics about earnings during a time
type EarningsSeries struct {
	Data []SeriesItem `json:"data"`
}

// SessionsSeries represents data series metrics about started sessions during a time
type SessionsSeries struct {
	Data []SeriesItem `json:"data"`
}

// TransferredDataSeries represents data series metrics about transferred bytes during a time
type TransferredDataSeries struct {
	Data []SeriesItem `json:"data"`
}

// ActivityStats represent a  provider activity stats
type ActivityStats struct {
	Online float64 `json:"online_percent"`
	Active float64 `json:"active_percent"`
}

// QualityInfo represents a provider quality info.
type QualityInfo struct {
	Quality float64 `json:"quality"`
}

// EarningsPerService represents information about earnings per service
type EarningsPerService struct {
	EarningsPublic        string `json:"public"`
	EarningsVPN           string `json:"data_transfer"`
	EarningsScraping      string `json:"scraping"`
	EarningsDVPN          string `json:"dvpn"`
	EarningsTotal         string `json:"total"`
	TotalEarningsPublic   string `json:"total_public"`
	TotalEarningsVPN      string `json:"total_data_transfer"`
	TotalEarningsScraping string `json:"total_scraping"`
	TotalEarningsDVPN     string `json:"total_dvpn"`
}

// Sessions retrieves and resolved monitoring status from quality oracle
func (m *StatsTracker) Sessions(rangeTime string) ([]SessionItem, error) {
	id, ok := m.currentIdentity.GetUnlockedIdentity()
	if ok {
		return m.providerSessionsList(id, rangeTime)
	}

	return []SessionItem{}, errIdentityNotFound
}

// TransferredData retrieves and resolved total traffic served by the provider
func (m *StatsTracker) TransferredData(rangeTime string) (TransferredData, error) {
	id, ok := m.currentIdentity.GetUnlockedIdentity()
	if ok {
		return m.providerTransferredData(id, rangeTime)
	}

	return TransferredData{}, errIdentityNotFound
}

// SessionsCount retrieves and resolved numbers of sessions
func (m *StatsTracker) SessionsCount(rangeTime string) (SessionsCount, error) {
	id, ok := m.currentIdentity.GetUnlockedIdentity()
	if ok {
		return m.providerSessionsCount(id, rangeTime)
	}

	return SessionsCount{}, errIdentityNotFound
}

// ConsumersCount retrieves and resolved numbers of consumers server during period of time
func (m *StatsTracker) ConsumersCount(rangeTime string) (ConsumersCount, error) {
	id, ok := m.currentIdentity.GetUnlockedIdentity()
	if ok {
		return m.providerConsumersCount(id, rangeTime)
	}

	return ConsumersCount{}, errIdentityNotFound
}

// EarningsSeries retrieves and resolved earnings data series metrics during a time range
func (m *StatsTracker) EarningsSeries(rangeTime string) (EarningsSeries, error) {
	id, ok := m.currentIdentity.GetUnlockedIdentity()
	if ok {
		return m.providerEarningsSeries(id, rangeTime)
	}

	return EarningsSeries{}, errIdentityNotFound
}

// SessionsSeries retrieves and resolved sessions data series metrics during a time range
func (m *StatsTracker) SessionsSeries(rangeTime string) (SessionsSeries, error) {
	id, ok := m.currentIdentity.GetUnlockedIdentity()
	if ok {
		return m.providerSessionsSeries(id, rangeTime)
	}

	return SessionsSeries{}, errIdentityNotFound
}

// TransferredDataSeries retrieves and resolved transferred bytes data series metrics during a time range
func (m *StatsTracker) TransferredDataSeries(rangeTime string) (TransferredDataSeries, error) {
	id, ok := m.currentIdentity.GetUnlockedIdentity()
	if ok {
		return m.providerTransferredDataSeries(id, rangeTime)
	}

	return TransferredDataSeries{}, errIdentityNotFound
}

// ProviderQuality retrieves and resolved provider quality
func (m *StatsTracker) ProviderQuality() (QualityInfo, error) {

	id, ok := m.currentIdentity.GetUnlockedIdentity()
	if ok {
		return m.providerQuality(id)
	}
	return QualityInfo{}, errIdentityNotFound
}

// ProviderActivityStats retrieves and resolved provider activity stats
func (m *StatsTracker) ProviderActivityStats() (ActivityStats, error) {
	id, ok := m.currentIdentity.GetUnlockedIdentity()
	if ok {
		return m.providerActivityStats(id)
	}

	return ActivityStats{}, errIdentityNotFound
}

// EarningsPerService retrieves and resolved earnings per service type
func (m *StatsTracker) EarningsPerService() (EarningsPerService, error) {
	id, ok := m.currentIdentity.GetUnlockedIdentity()
	if ok {
		return m.providerServiceEarnings(id)
	}

	return EarningsPerService{}, errIdentityNotFound
}
