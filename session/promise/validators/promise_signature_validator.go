package validators

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/mysteriumnetwork/node/session/promise"
	"github.com/mysteriumnetwork/payments/promises"
)

type IssuedPromiseValidator struct {
	consumer common.Address
	receiver common.Address
	issuer   common.Address
}

func (ipv *IssuedPromiseValidator) Validate(promiseMsg promise.PromiseMessage) bool {
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
