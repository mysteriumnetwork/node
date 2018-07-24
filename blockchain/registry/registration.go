package registry

import (
	"github.com/MysteriumNetwork/payments/registry"
	"github.com/MysteriumNetwork/payments/registry/generated"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
)

type IdentityRegistry interface {
	IsRegistered(identity common.Address) (bool, error)
}

type RegistrationDataProvider interface {
	ProvideRegistrationData(identity common.Address) (*registry.RegistrationData, error)
}

type keystoreRegistrationDataProvider struct {
	ks *keystore.KeyStore
}

func (kpg *keystoreRegistrationDataProvider) ProvideRegistrationData(identity common.Address) (*registry.RegistrationData, error) {
	identityHolder := registry.FromKeystore(kpg.ks, identity)

	return registry.CreateRegistrationData(identityHolder)
}

func NewRegistrationDataProvider(ks *keystore.KeyStore) RegistrationDataProvider {
	return &keystoreRegistrationDataProvider{
		ks: ks,
	}
}

func NewIdentityRegistry(contractCaller bind.ContractCaller, registryAddress common.Address) (IdentityRegistry, error) {
	contract, err := generated.NewIdentityRegistryCaller(registryAddress, contractCaller)
	if err != nil {
		return nil, err
	}

	return &generated.IdentityRegistryCallerSession{
		Contract: contract,
		CallOpts: bind.CallOpts{
			Pending: false, //we want to find out true registration status - not pending transactions
		},
	}, nil
}
