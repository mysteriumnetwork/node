package validators

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/session/promise"
	"github.com/mysteriumnetwork/payments/promises"
)

// IssuedPromiseValidator validates issued promises
type IssuedPromiseValidator struct {
	consumer common.Address
	receiver common.Address
	issuer   common.Address
}

// NewIssuedPromiseValidator return a new instance of IssuedPromiseValidator
func NewIssuedPromiseValidator(consumer, receiver, issuer identity.Identity) *IssuedPromiseValidator {
	return &IssuedPromiseValidator{
		consumer: common.HexToAddress(consumer.Address),
		receiver: common.HexToAddress(receiver.Address),
		issuer:   common.HexToAddress(issuer.Address),
	}
}

// Validate checks if the issued promise is valid or not
func (ipv *IssuedPromiseValidator) Validate(promiseMsg promise.Message) bool {
	issuedPromise := promises.IssuedPromise{
		Promise: promises.Promise{
			Extra: promise.ExtraData{
				ConsumerAddress: ipv.consumer,
			},
			Amount:   int64(promiseMsg.Amount),
			SeqNo:    int64(promiseMsg.SequenceID),
			Receiver: ipv.receiver,
		},
		IssuerSignature: common.FromHex(promiseMsg.Signature),
	}

	issuer, err := issuedPromise.IssuerAddress()
	if err != nil {
		return false
	}
	return issuer == ipv.issuer
}
