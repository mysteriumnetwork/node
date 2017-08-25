package dto

import (
	dto_discovery "github.com/mysterium/node/service_discovery/dto"
)

type ServiceDefinition struct {
	// Approximate information on location where the service is provided from
	Location dto_discovery.Location

	// Approximate information on location where the tunnelled traffic will originate from
	LocationOriginate dto_discovery.Location

	// Available per session bandwidth
	SessionBandwidth Bandwidth
}

func (service ServiceDefinition) GetLocation() dto_discovery.Location {
	return service.Location
}
