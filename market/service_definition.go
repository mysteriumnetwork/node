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

package market

import "encoding/json"

// ServiceDefinition interface is interface for all service definition types
type ServiceDefinition interface {
	GetLocation() Location
}

// UnsupportedServiceDefinition represents unknown or unsupported service definition returned by deserializer
type UnsupportedServiceDefinition struct {
}

// GetLocation always panics on unsupported service types
func (UnsupportedServiceDefinition) GetLocation() Location {
	//no location available - should never be called
	panic("not supported")
}

var _ ServiceDefinition = UnsupportedServiceDefinition{}

// ServiceDefinitionUnserializer defines function to register for concrete service definition
type ServiceDefinitionUnserializer func(*json.RawMessage) (ServiceDefinition, error)

// service definition unserializer registry
//TODO same idea as for contact global map
var serviceDefinitionMap = make(map[string]ServiceDefinitionUnserializer)

// RegisterServiceDefinitionUnserializer registers deserializer for specified service definition
func RegisterServiceDefinitionUnserializer(serviceType string, unserializer ServiceDefinitionUnserializer) {
	serviceDefinitionMap[serviceType] = unserializer
}
func unserializeServiceDefinition(serviceType string, message *json.RawMessage) ServiceDefinition {
	method, ok := serviceDefinitionMap[serviceType]
	if !ok {
		return UnsupportedServiceDefinition{}
	}
	sd, err := method(message)
	if err != nil {
		return UnsupportedServiceDefinition{}
	}
	return sd
}
