package location

import (
	"github.com/oschwald/geoip2-golang"
	"github.com/pkg/errors"
	"net"
)

// Database downloaded from http://dev.maxmind.com/geoip/geoip2/geolite2/
const Database = "data/GeoLite2-Country.mmdb"

func DetectCountry(ip string) (string, error) {
	return DetectCountryWithDatabase(ip, Database)
}

func DetectCountryWithDatabase(ip, database string) (string, error) {
	db, err := geoip2.Open(database)
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
