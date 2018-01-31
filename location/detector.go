package location

import (
	"github.com/oschwald/geoip2-golang"
	"github.com/pkg/errors"
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
	countryRecord, err := db.Country(ipObject)
	if err != nil {
		return "", err
	}
	country := countryRecord.Country.IsoCode
	if country == "" {
		return "", errors.New("country was not found")
	}
	return country, nil
}
