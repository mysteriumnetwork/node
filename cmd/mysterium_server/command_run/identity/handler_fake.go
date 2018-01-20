package identity

import (
	"errors"
	"github.com/mysterium/node/identity"
)

type handlerFake struct {
	LastAddress string
}

func (hf *handlerFake) UseExisting(address string) (identity.Identity, error) {
	return identity.Identity{Address: address}, nil
}

func (hf *handlerFake) UseLast() (id identity.Identity, err error) {
	if hf.LastAddress != "" {
		id = identity.Identity{Address: hf.LastAddress}
	} else {
		err = errors.New("no last identity")
	}
	return
}

func (hf *handlerFake) UseNew(passphrase string) (identity.Identity, error) {
	return identity.Identity{Address: "new"}, nil
}
