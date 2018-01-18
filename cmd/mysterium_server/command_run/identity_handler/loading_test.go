package identity_handler

import (
	"testing"
	"github.com/stretchr/testify/assert"
)

func Test_LoadIdentityExisting(t *testing.T) {
	identityHandler := &identityHandlerFake{}
	id, err := LoadIdentity(identityHandler, "existing", "")
	assert.Equal(t, "existing", id.Address)
	assert.Nil(t, err)
}

func Test_LoadIdentityLast(t *testing.T) {
	identityHandler := &identityHandlerFake{LastAddress: "last"}
	id, err := LoadIdentity(identityHandler, "", "")
	assert.Equal(t, "last", id.Address)
	assert.Nil(t, err)
}

func Test_LoadIdentityNew(t *testing.T) {
	identityHandler := &identityHandlerFake{}
	id, err := LoadIdentity(identityHandler, "", "")
	assert.Equal(t, "new", id.Address)
	assert.Nil(t, err)
}

