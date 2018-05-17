package location

import (
	"github.com/mysterium/node/ip"
)

type detector struct {
	ipResolver       ip.Resolver
	locationResolver Resolver
}

// NewDetector constructs Detector
func NewDetector(ipResolver ip.Resolver, databasePath string) Detector {
	return NewDetectorWithLocationResolver(ipResolver, NewResolver(databasePath))
}

// NewDetectorWithLocationResolver constructs Detector
func NewDetectorWithLocationResolver(ipResolver ip.Resolver, locationResolver Resolver) Detector {
	return &detector{
		ipResolver:       ipResolver,
		locationResolver: locationResolver,
	}
}

// Maps current ip to country
func (d *detector) DetectLocation() (Location, error) {
	ip, err := d.ipResolver.GetPublicIP()
	if err != nil {
		return Location{}, err
	}

	country, err := d.locationResolver.ResolveCountry(ip)
	if err != nil {
		return Location{}, err
	}

	location := Location{Country: country, IP: ip}
	return location, nil
}
