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

package quality

// ServiceMetricsResponse represents response from the quality oracle service
type ServiceMetricsResponse struct {
	Connects []ConnectMetric `json:"connects"`
}

// ConnectMetric represents a proposal with quality info
type ConnectMetric struct {
	ProposalID   ProposalID   `json:"proposalId"`
	ConnectCount ConnectCount `json:"connectCount"`
}

// ProposalID represents the struct used to uniquely identify proposals
type ProposalID struct {
	ProviderID  string `json:"providerId" example:"0x286f0e9eb943eca95646bf4933698856579b096e"`
	ServiceType string `json:"serviceType" example:"openvpn"`
}

// ConnectCount represents the connection count statistics
type ConnectCount struct {
	Success int `json:"success" example:"100" format:"int64"`
	Fail    int `json:"fail" example:"50" format:"int64"`
	Timeout int `json:"timeout" example:"10" format:"int64"`
}
