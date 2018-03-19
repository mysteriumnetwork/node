package location

import (
	"errors"
	"github.com/oschwald/geoip2-golang"
	"net"
)

type resolver struct {
	databasePath string
}

// NewResolver returns Resolver which uses country database
func NewResolver(databasePath string) Resolver {
	return &resolver{
		databasePath: databasePath,
	}
}

// ResolveCountry maps given ip to country
func (r *resolver) ResolveCountry(ip string) (string, error) {
	db, err := geoip2.Open(r.databasePath)
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
