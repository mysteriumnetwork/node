package promise

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/mysteriumnetwork/payments/promises"
)

type ExtraData struct {
	ConsumerAddress common.Address
}

func (extra ExtraData) Hash() []byte {
	return crypto.Keccak256(extra.ConsumerAddress.Bytes())
}

var _ promises.ExtraData = ExtraData{}
