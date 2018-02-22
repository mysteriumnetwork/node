package identity

// IdentityCacheInterface allows caching single identity
type IdentityCacheInterface interface {
	GetIdentity() (identity Identity, err error)
	StoreIdentity(identity Identity) error
}
