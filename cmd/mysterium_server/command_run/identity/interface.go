package identity

import "github.com/mysterium/node/identity"

type HandlerInterface interface {
	UseExisting(address string) (identity.Identity, error)
	UseLast() (identity.Identity, error)
	UseNew(passphrase string) (identity.Identity, error)
}
