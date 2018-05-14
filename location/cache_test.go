package location

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestLocationCacheGetLocationWithoutCache(t *testing.T) {
	cache := cache{}
	assert.Equal(t, Location{}, cache.Get())
}

func TestLocationCacheGetsCachedLocation(t *testing.T) {
	locationExpected := Location{"100.100.100.100", "country"}

	cache := cache{locationExpected}
	assert.Equal(t, locationExpected, cache.Get())
}

func TestLocationCacheSetsLocation(t *testing.T) {
	locationExpected := Location{"100.100.100.100", "country"}

	cache := cache{}
	cache.Set(locationExpected)
	assert.Equal(t, locationExpected, cache.location)
}
