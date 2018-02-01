package identity

import "github.com/mysterium/node/identity"

// Selector selects the identity
type Selector func() (identity.Identity, error)

// LoadIdentity chooses which identity to use and invokes it using identityHandler
func LoadIdentity(identityHandler HandlerInterface, keyOption, passphrase string) (identity.Identity, error) {
	if len(keyOption) > 0 {
		return identityHandler.UseExisting(keyOption, passphrase)
	}

	if id, err := identityHandler.UseLast(passphrase); err == nil {
		return id, err
	}

	return identityHandler.UseNew(passphrase)
}
