package identity

type IdentityManagerInterface interface {
	CreateNewIdentity(passphrase string) (Identity, error)
	GetIdentities() []Identity
	GetIdentity(address string) (Identity, error)
	HasIdentity(address string) bool
	Unlock(address string, passphrase string) error
}
