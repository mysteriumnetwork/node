package registration

import (
	"github.com/MysteriumNetwork/payments/registry"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
)

type ProofGenerator interface {
	GenerateProofForIdentity(identity common.Address) (*registry.ProofOfIdentity, error)
}

type keystoreProofGenerator struct {
	ks *keystore.KeyStore
}

func (kpg *keystoreProofGenerator) GenerateProofForIdentity(identity common.Address) (*registry.ProofOfIdentity, error) {
	identityHolder := registry.FromKeystoreIdentity(kpg.ks, identity)

	return registry.CreateProofOfIdentity(identityHolder)
}

func NewProofGenerator(ks *keystore.KeyStore) ProofGenerator {
	return &keystoreProofGenerator{
		ks: ks,
	}
}
