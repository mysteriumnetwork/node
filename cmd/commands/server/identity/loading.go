package identity

import "github.com/mysterium/node/identity"

// Selector selects the identity
type Selector func() (identity.Identity, error)

// LoadIdentity chooses which identity to use and invokes it using identityHandler
func LoadIdentity(identityHandler HandlerInterface, identityOption, passphrase string) (identity.Identity, error) {
	if len(identityOption) > 0 {
		return identityHandler.UseExisting(identityOption, passphrase)
	}

	if id, err := identityHandler.UseLast(passphrase); err == nil {
		return id, err
	}

	return identityHandler.UseNew(passphrase)
}
