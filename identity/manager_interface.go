package identity

type IdentityManagerInterface interface {
	CreateNewIdentity(string) (Identity, error)
	GetIdentities() []Identity
	GetIdentity(string) (Identity, error)
	HasIdentity(string) bool
}
