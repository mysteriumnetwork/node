package identity

type identityCacheFake struct {
	identity Identity
}

func NewIdentityCacheFake() IdentityCacheInterface {
	return &identityCacheFake{}
}

func (icf *identityCacheFake) GetIdentity() (identity Identity, err error) {
	return icf.identity, nil
}

func (icf *identityCacheFake) StoreIdentity(identity Identity) error {
	icf.identity = identity
	return nil
}
