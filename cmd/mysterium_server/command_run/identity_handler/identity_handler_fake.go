package identity_handler

import (
	"github.com/mysterium/node/identity"
	"errors"
)

type identityHandlerFake struct {
	LastAddress string
}

func (ihf *identityHandlerFake) UseExisting(address, passphrase string) (identity.Identity, error) {
	return identity.Identity{Address: address}, nil
}

func (ihf *identityHandlerFake) UseLast(passphrase string) (id identity.Identity, err error) {
	if ihf.LastAddress != "" {
		id = identity.Identity{Address: ihf.LastAddress}
	} else {
		err = errors.New("no last identity")
	}
	return
}

func (ihf *identityHandlerFake) UseNew(passphrase string) (identity.Identity, error) {
	return identity.Identity{Address: "new"}, nil
}

