package identity_handler

import "github.com/mysterium/node/identity"

type IdentityHandlerInterface interface {
	UseExisting(address string) (identity.Identity, error)
	UseLast() (identity.Identity, error)
	UseNew(passphrase string) (identity.Identity, error)
}
