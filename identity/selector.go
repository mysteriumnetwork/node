package identity

import (
	"errors"
	"github.com/mysterium/node/service_discovery/dto"
)

const DEFAULT_PASSPHRASE = ""

func SelectIdentity(dir string, id string) (identity *dto.Identity, err error) {
	manager := NewIdentityManager(dir)

	if len(id) > 0 {
		if !manager.HasIdentity(id) {
			return identity, errors.New("identity doesn't exist")
		}

		identity = manager.GetIdentity(id)
		return identity, nil
	}

	identity, err = manager.CreateNewIdentity(DEFAULT_PASSPHRASE)
	return identity, err
}
