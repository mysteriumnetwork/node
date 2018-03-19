package location

import (
	"github.com/mysterium/node/ip"
)

type detector struct {
	ipResolver ip.Resolver
	locationResolver Resolver
}

// NewDetector constructs Detector
func NewDetector(ipResolver ip.Resolver, databasePath string) Detector {
	return &detector{
		ipResolver: ipResolver,
		locationResolver: NewResolver(databasePath),
	}
}

// NewDetectorWithLocationResolver constructs Detector
func NewDetectorWithLocationResolver(ipResolver ip.Resolver, locationResolver Resolver) Detector {
	return &detector{
		ipResolver: ipResolver,
		locationResolver: locationResolver,
	}
}

// Maps current ip to country
func (d *detector) DetectCountry() (string, error) {
	ip, err := d.ipResolver.GetPublicIP()
	if err != nil {
		return "", err
	}

	country, err := d.locationResolver.ResolveCountry(ip)
	if err != nil {
		return "", err
	}

	return country, nil
}
