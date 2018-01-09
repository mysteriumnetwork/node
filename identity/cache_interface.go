package identity

type IdentityCacheInterface interface {
	GetIdentity() (identity Identity, err error)
	StoreIdentity(identity Identity) error
}
