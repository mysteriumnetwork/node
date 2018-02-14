package location

import (
	"errors"
	"github.com/oschwald/geoip2-golang"
	"net"
)

type detector struct {
	databasePath string
}

// NewDetector returns Detector which uses country database
func NewDetector(databasePath string) *detector {
	return &detector{
		databasePath: databasePath,
	}
}

// DetectCountry maps given ip to country
func (d *detector) DetectCountry(ip string) (string, error) {
	db, err := geoip2.Open(d.databasePath)
	if err != nil {
		return "", err
	}
	defer db.Close()

	ipObject := net.ParseIP(ip)
	if ipObject == nil {
		return "", errors.New("failed to parse IP")
	}

	countryRecord, err := db.Country(ipObject)
	if err != nil {
		return "", err
	}

	country := countryRecord.Country.IsoCode
	if country == "" {
		country = countryRecord.RegisteredCountry.IsoCode
	}

	return country, nil
}
