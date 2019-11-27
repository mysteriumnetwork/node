package pingpong

import (
	"errors"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/mysteriumnetwork/payments/bindings"
	"github.com/rs/zerolog/log"
)

type blockchain interface {
	GetAccountantFee(accountantAddress common.Address) (uint16, error)
	IsRegisteredAsProvider(accountantAddress, registryAddress, addressToCheck common.Address) (bool, error)
	GetProviderChannel(accountantAddress common.Address, addressToCheck common.Address) (ProviderChannel, error)
	IsRegistered(registryAddress, addressToCheck common.Address) (bool, error)
	SubscribeToPromiseSettledEvent(providerID, accountantID common.Address) (sink chan *bindings.AccountantImplementationPromiseSettled, cancel func(), err error)
}

// BlockchainWithRetries takes in the plain blockchain implementation and exposes methods that will retry the underlying bc methods before giving up.
// This is required as the ethereum client will occasionally spit a TLS error if running for prolonged periods of time.
type BlockchainWithRetries struct {
	delay      time.Duration
	maxRetries int
	bc         blockchain
	stop       chan struct{}
	once       sync.Once
}

// ErrStopped represents an error when a call is interrupted
var ErrStopped = errors.New("call stopped")

// NewBlockchainWithRetries returns a new instance of blockchain with retries
func NewBlockchainWithRetries(bc blockchain, delay time.Duration, maxRetries int) *BlockchainWithRetries {
	return &BlockchainWithRetries{
		bc:         bc,
		delay:      delay,
		maxRetries: maxRetries,
	}
}

// GetAccountantFee fetches the accountant fee from blockchain
func (bwr *BlockchainWithRetries) GetAccountantFee(accountantAddress common.Address) (uint16, error) {
	for i := 0; i < bwr.maxRetries; i++ {
		res, err := bwr.bc.GetAccountantFee(accountantAddress)
		if err == nil {
			return res, nil
		}
		if i+1 == bwr.maxRetries {
			return 0, err
		}

		log.Warn().Err(err).Msgf("retry %v of %v", i+1, bwr.maxRetries)
		select {
		case <-bwr.stop:
			return 0, ErrStopped
		case <-time.After(bwr.delay):
		}
	}
	return bwr.bc.GetAccountantFee(accountantAddress)
}

// IsRegisteredAsProvider checks if the provider is registered with the accountant properly
func (bwr *BlockchainWithRetries) IsRegisteredAsProvider(accountantAddress, registryAddress, addressToCheck common.Address) (bool, error) {
	for i := 0; i < bwr.maxRetries; i++ {
		res, err := bwr.bc.IsRegisteredAsProvider(accountantAddress, registryAddress, addressToCheck)
		if err == nil {
			return res, nil
		}
		if i+1 == bwr.maxRetries {
			return res, err
		}

		log.Warn().Err(err).Msgf("retry %v of %v", i+1, bwr.maxRetries)
		select {
		case <-bwr.stop:
			return false, ErrStopped
		case <-time.After(bwr.delay):
		}
	}
	return bwr.bc.IsRegisteredAsProvider(accountantAddress, registryAddress, addressToCheck)
}

// GetProviderChannel returns the provider channel
func (bwr *BlockchainWithRetries) GetProviderChannel(accountantAddress, addressToCheck common.Address) (ProviderChannel, error) {
	for i := 0; i < bwr.maxRetries; i++ {
		res, err := bwr.bc.GetProviderChannel(accountantAddress, addressToCheck)
		if err == nil {
			return res, nil
		}
		if i+1 == bwr.maxRetries {
			return res, err
		}

		log.Warn().Err(err).Msgf("retry %v of %v", i+1, bwr.maxRetries)
		select {
		case <-bwr.stop:
			return ProviderChannel{}, ErrStopped
		case <-time.After(bwr.delay):
		}
	}
	return bwr.bc.GetProviderChannel(accountantAddress, addressToCheck)
}

// SubscribeToPromiseSettledEvent subscribes to promise settled events
func (bwr *BlockchainWithRetries) SubscribeToPromiseSettledEvent(providerID, accountantID common.Address) (sink chan *bindings.AccountantImplementationPromiseSettled, cancel func(), err error) {
	for i := 0; i < bwr.maxRetries; i++ {
		sink, cancel, err := bwr.bc.SubscribeToPromiseSettledEvent(providerID, accountantID)
		if err == nil {
			return sink, cancel, nil
		}
		if i+1 == bwr.maxRetries {
			return nil, nil, err
		}

		log.Warn().Err(err).Msgf("retry %v of %v", i+1, bwr.maxRetries)
		select {
		case <-bwr.stop:
			return sink, cancel, ErrStopped
		case <-time.After(bwr.delay):
		}
	}
	return bwr.bc.SubscribeToPromiseSettledEvent(providerID, accountantID)
}

// IsRegistered checks wether the given identity is registered or not
func (bwr *BlockchainWithRetries) IsRegistered(registryAddress, addressToCheck common.Address) (bool, error) {
	for i := 0; i < bwr.maxRetries; i++ {
		res, err := bwr.bc.IsRegistered(registryAddress, addressToCheck)
		if err == nil {
			return res, nil
		}
		if i+1 == bwr.maxRetries {
			return res, err
		}

		log.Warn().Err(err).Msgf("retry %v of %v", i+1, bwr.maxRetries)
		select {
		case <-bwr.stop:
			return res, ErrStopped
		case <-time.After(bwr.delay):
		}
	}
	return bwr.bc.IsRegistered(registryAddress, addressToCheck)
}

// Stop stops the blockhain with retries aborting any waits for retries
func (bwr *BlockchainWithRetries) Stop() {
	bwr.once.Do(func() {
		close(bwr.stop)
	})
}
