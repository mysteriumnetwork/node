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
	country, err := detector.DetectCountry()
	assert.Equal(t, "US", country)
	assert.NoError(t, err)
}

func TestWithIpResolverFailing(t *testing.T) {
	ipErr := errors.New("ip resolver error")
	ipResolver := ip.NewFailingFakeResolver(ipErr)
	detector := NewDetectorWithLocationResolver(ipResolver, NewResolverFake(""))
	country, err := detector.DetectCountry()
	assert.EqualError(t, ipErr, err.Error())
	assert.Equal(t, "", country)
}

func TestWithLocationResolverFailing(t *testing.T) {
	ipResolver := ip.NewFakeResolver("")
	locationErr := errors.New("location resolver error")
	locationResolver := NewFailingResolverFake(locationErr)
	detector := NewDetectorWithLocationResolver(ipResolver, locationResolver)
	country, err := detector.DetectCountry()
	assert.EqualError(t, locationErr, err.Error())
	assert.Equal(t, "", country)
}