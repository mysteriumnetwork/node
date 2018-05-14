package location

import (
	"errors"
	"github.com/mysterium/node/ip"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNewDetector(t *testing.T) {
	detector := NewDetector(ip.NewFakeResolver("8.8.8.8"), NewResolverFake("US"))

	location, err := detector.DetectLocation()
	assert.Equal(t, "US", location.Country)
	assert.Equal(t, "8.8.8.8", location.IP)
	assert.NoError(t, err)
}

func TestWithIpResolverFailing(t *testing.T) {
	ipErr := errors.New("ip resolver error")
	ipResolver := ip.NewFailingFakeResolver(ipErr)
	detector := NewDetector(ipResolver, NewResolverFake(""))
	location, err := detector.DetectLocation()
	assert.EqualError(t, ipErr, err.Error())
	assert.Equal(t, Location{}, location)
}

func TestWithLocationResolverFailing(t *testing.T) {
	ipResolver := ip.NewFakeResolver("")
	locationErr := errors.New("location resolver error")
	locationResolver := NewFailingResolverFake(locationErr)
	detector := NewDetector(ipResolver, locationResolver)
	location, err := detector.DetectLocation()
	assert.EqualError(t, locationErr, err.Error())
	assert.Equal(t, Location{}, location)
}
