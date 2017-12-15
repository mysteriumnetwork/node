package identity

import (
	"testing"
	"github.com/mysterium/node/service_discovery/dto"
	"github.com/stretchr/testify/assert"
	"os"
)

var file = "/tmp/cache.json"

func TestIdentityCache_StoreIdentity(t *testing.T) {
	identity := dto.Identity("0x000000000000000000000000000000000000000A")
	cache := identityCache{
		File: file,
	}

	err := cache.StoreIdentity(identity)
	assert.Nil(t, err)
}

func Test_IdentityCacheGetIdentity(t *testing.T) {
	identity := dto.Identity("0x000000000000000000000000000000000000000A")
	cache := identityCache{
		File: file,
	}

	err := cache.StoreIdentity(identity)
	assert.Nil(t, err)
	assert.Equal(t, cache.GetIdentity(), identity)
}

func Test_cacheExists(t *testing.T) {
	identity := dto.Identity("0x000000000000000000000000000000000000000A")
	cache := identityCache{
		File: file,
	}

	err := cache.StoreIdentity(identity)
	assert.Nil(t, err)

	assert.True(t, cache.cacheExists())

	_, err = os.Stat(file)
	assert.True(t, err == nil && !os.IsNotExist(err))
}
