package location

import (
	"github.com/mysterium/node/ip"
)

type detector struct {
	ipResolver       ip.Resolver
	locationResolver Resolver
}

// NewDetector constructs Detector
func NewDetector(ipResolver ip.Resolver, locationResolver Resolver) Detector {
	return &detector{
		ipResolver:       ipResolver,
		locationResolver: locationResolver,
	}
}

// Maps current ip to country
func (d *detector) DetectLocation() (location Location, err error) {
	location.IP, err = d.ipResolver.GetPublicIP()
	if err != nil {
		return
	}

	location.Country, err = d.locationResolver.ResolveCountry(location.IP)
	return
}
