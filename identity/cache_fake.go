package identity

type identityCacheFake struct {
	identity Identity
}

// NewIdentityCacheFake creates and returns fake identity cache
func NewIdentityCacheFake() IdentityCacheInterface {
	return &identityCacheFake{}
}

// GetIdentity returns mocked identity
func (icf *identityCacheFake) GetIdentity() (identity Identity, err error) {
	return icf.identity, nil
}

// StoreIdentity saves identity to be retrieved later
func (icf *identityCacheFake) StoreIdentity(identity Identity) error {
	icf.identity = identity
	return nil
}
