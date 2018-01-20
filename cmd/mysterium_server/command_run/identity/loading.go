package identity

import "github.com/mysterium/node/identity"

type IdentitySelector func() (identity.Identity, error)

// LoadIdentity selects and unlocks identity
func LoadIdentity(identitySelector IdentitySelector, identityManager identity.IdentityManagerInterface, passphrase string) (identity.Identity, error) {
	id, err := identitySelector()

	if err != nil {
		return id, err
	}

	if err = identityManager.Unlock(id.Address, passphrase); err != nil {
		return id, err
	}

	return id, nil
}

//SelectIdentity selects lastUsed identity or creates and unlocks new one if keyOption is not present
func SelectIdentity(identityHandler HandlerInterface, keyOption, passphrase string) (identity.Identity, error) {
	if len(keyOption) > 0 {
		return identityHandler.UseExisting(keyOption)
	}

	if id, err := identityHandler.UseLast(); err == nil {
		return id, err
	}

	return identityHandler.UseNew(passphrase)
}
