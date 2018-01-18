package identity_handler

import "github.com/mysterium/node/identity"

type identityHandlerInterface interface {
	UseExisting(address, passphrase string) (id identity.Identity, err error)
	UseLast(passphrase string) (identity identity.Identity, err error)
	UseNew(passphrase string) (id identity.Identity, err error)
}
