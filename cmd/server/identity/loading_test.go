package identity

import (
	"errors"
	"github.com/mysterium/node/identity"
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_LoadIdentityExisting(t *testing.T) {
	identityHandler := &handlerFake{}
	id, err := LoadIdentity(identityHandler, "existing", "")
	assert.Equal(t, "existing", id.Address)
	assert.Nil(t, err)
}

func Test_LoadIdentityLast(t *testing.T) {
	identityHandler := &handlerFake{LastAddress: "last"}
	id, err := LoadIdentity(identityHandler, "", "")
	assert.Equal(t, "last", id.Address)
	assert.Nil(t, err)
}

func Test_LoadIdentityNew(t *testing.T) {
	identityHandler := &handlerFake{}
	id, err := LoadIdentity(identityHandler, "", "")
	assert.Equal(t, "new", id.Address)
	assert.Nil(t, err)
}
