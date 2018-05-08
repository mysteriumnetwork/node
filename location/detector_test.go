package location

import (
	"testing"
	"errors"
	"github.com/mysterium/node/ip"
	"github.com/stretchr/testify/assert"
)

func TestNewDetector(t *testing.T) {
	ipResolver := ip.NewFakeResolver("8.8.8.8")
	detector := NewDetector(ipResolver, "../bin/client_package/config/GeoLite2-Country.mmdb")
	location, err := detector.DetectLocation()
	assert.Equal(t, "US", location.Country)
	assert.Equal(t, "8.8.8.8", location.IP)
	assert.NoError(t, err)
}

func TestWithIpResolverFailing(t *testing.T) {
	ipErr := errors.New("ip resolver error")
	ipResolver := ip.NewFailingFakeResolver(ipErr)
	detector := NewDetectorWithLocationResolver(ipResolver, NewResolverFake(""))
	location, err := detector.DetectLocation()
	assert.EqualError(t, ipErr, err.Error())
	assert.Equal(t, Location{}, location)
}

func TestWithLocationResolverFailing(t *testing.T) {
	ipResolver := ip.NewFakeResolver("")
	locationErr := errors.New("location resolver error")
	locationResolver := NewFailingResolverFake(locationErr)
	detector := NewDetectorWithLocationResolver(ipResolver, locationResolver)
	location, err := detector.DetectLocation()
	assert.EqualError(t, locationErr, err.Error())
	assert.Equal(t, Location{}, location)
}
