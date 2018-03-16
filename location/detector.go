package location

import (
	"github.com/mysterium/node/ip"
)

type detector struct {
	ipResolver ip.Resolver
	locationResolver Resolver
}

func NewDetector(ipResolver ip.Resolver, databasePath string) *detector {
	return &detector{
		ipResolver: ipResolver,
		locationResolver: NewResolver(databasePath),
	}
}

func NewDetectorWithLocationResolver(ipResolver ip.Resolver, locationResolver Resolver) *detector {
	return &detector{
		ipResolver: ipResolver,
		locationResolver: locationResolver,
	}
}

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