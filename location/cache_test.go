package location

import (
	"errors"
	"github.com/mysterium/node/ip"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestLocationCacheFirstCall(t *testing.T) {
	ipResolver := ip.NewFakeResolver("100.100.100.100")
	locationResolver := NewResolverFake("country")
	locationDetector := NewDetectorWithLocationResolver(ipResolver, locationResolver)
	locationCache := NewLocationCache(locationDetector)
	location := locationCache.Get()
	assert.Equal(t, Location{}, location)
}

func TestLocationCacheFirstSecondCalls(t *testing.T) {
	ipResolver := ip.NewFakeResolver("100.100.100.100")
	locationResolver := NewResolverFake("country")
	locationDetector := NewDetectorWithLocationResolver(ipResolver, locationResolver)
	locationCache := NewLocationCache(locationDetector)
	location, err := locationCache.RefreshAndGet()
	assert.Equal(t, "country", location.Country)
	assert.Equal(t, "100.100.100.100", location.IP)
	assert.NoError(t, err)

	locationSecondCall := locationCache.Get()
	assert.Equal(t, location, locationSecondCall)
}

func TestLocationCacheWithError(t *testing.T) {
	ipResolver := ip.NewFakeResolver("")
	locationErr := errors.New("location resolver error")
	locationResolver := NewFailingResolverFake(locationErr)
	locationDetector := NewDetectorWithLocationResolver(ipResolver, locationResolver)
	locationCache := NewLocationCache(locationDetector)
	location, err := locationCache.RefreshAndGet()
	assert.EqualError(t, locationErr, err.Error())
	assert.Equal(t, Location{}, location)
}
