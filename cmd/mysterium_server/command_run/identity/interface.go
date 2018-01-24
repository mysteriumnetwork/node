package identity

import "github.com/mysterium/node/identity"

// HandlerInterface allows selecting identity to be used
type HandlerInterface interface {
	UseExisting(address, passphrase string) (identity.Identity, error)
	UseLast(passphrase string) (identity.Identity, error)
	UseNew(passphrase string) (identity.Identity, error)
}
