package location

import (
	"github.com/mysterium/node/ip"
)

type locationDetector struct {
	ipResolver ip.Resolver
	locationDetector Detector
}

func NewLocationDetector(ipResolver ip.Resolver, databasePath string) *locationDetector {
	return &locationDetector{
		ipResolver: ipResolver,
		locationDetector: NewDetector(databasePath),
	}
}

func (cd *locationDetector) DetectCountry() (string, error) {
	ip, err := cd.ipResolver.GetPublicIP()
	if err != nil {
		return "", err
	}

	country, err := cd.locationDetector.DetectCountry(ip)
	if err != nil {
		return "", err
	}

	return country, nil
}