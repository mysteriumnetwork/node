/*
 * Copyright (C) 2017 The "MysteriumNetwork/node" Authors.
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

package dto

// ServiceQualityResponse represents response from the quality oracle service
type ServiceQualityResponse struct {
	Connects []QualityConnects `json:"connects"`
}

// QualityConnects represents a single proposal connection quality description
type QualityConnects struct {
	Proposal     QualityProposal `json:"proposal"`
	CountAll     int             `json:"countAll"`
	CountSuccess int             `json:"countSuccess"`
	CountFail    int             `json:"countFail"`
	CountTimeout int             `json:"countTimeout"`
}

// QualityProposal represents a proposal definition from the quality oracle service
type QualityProposal struct {
	ID                int                `json:"id"`
	ProviderID        string             `json:"ProviderID"`
	ServiceType       string             `json:"ServiceType"`
	ServiceDefinition *ServiceDefinition `json:"ServiceDefinition,omitempty"`
}

// ServiceDefinition represents the service definition struct
type ServiceDefinition struct {
	LocationOriginate LocationOriginate `json:"locationOriginate"`
}

// LocationOriginate represents the location originate
type LocationOriginate struct {
	Country string `json:"country"`
}
