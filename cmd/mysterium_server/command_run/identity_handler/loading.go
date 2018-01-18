package identity_handler

import "github.com/mysterium/node/identity"

//LoadIdentity selects and unlocks lastUsed identity or creates and unlocks new one if keyOption is not present
func LoadIdentity(identityHandler identityHandlerInterface, keyOption, passphrase string) (id identity.Identity, err error) {
	if len(keyOption) > 0 {
		return identityHandler.UseExisting(keyOption, passphrase)
	}

	if id, err = identityHandler.UseLast(passphrase); err == nil {
		return id, err
	}

	return identityHandler.UseNew(passphrase)
}

