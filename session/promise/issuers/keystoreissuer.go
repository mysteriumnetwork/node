package issuers

import (
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/session/promise"
	payments_identity "github.com/mysteriumnetwork/payments/identity"
	"github.com/mysteriumnetwork/payments/promises"
)

// LocalIssuer issues signed promise by using identity.Signer (usually based on local keystore)
type LocalIssuer struct {
	paymentsSigner payments_identity.Signer
}

// NewLocalIssuer creates local issuer based on provided identity signer
func NewLocalIssuer(signer identity.Signer) *LocalIssuer {
	return &LocalIssuer{
		paymentsSigner: paymentsSignerAdapter{
			identitySigner: signer,
		},
	}
}

func (li LocalIssuer) Issue(promise promises.Promise) (promises.IssuedPromise, error) {

	signed, err := promises.SignByPayer(&promise, li.paymentsSigner)
	if err != nil {
		// TODO this looks ugly - align interface or discard pointers to structs?
		return promises.IssuedPromise{}, err
	}
	return *signed, nil
}

var _ promise.Issuer = LocalIssuer{}

// this is ugly adapter to make identity.Signer from node usable in payments package
// it's a bit confusing as both interfaces has the same name and method, but only params and return values differ
type paymentsSignerAdapter struct {
	identitySigner identity.Signer
}

func (psa paymentsSignerAdapter) Sign(data ...[]byte) ([]byte, error) {
	var message []byte
	for _, dataSlice := range data {
		message = append(message, dataSlice...)
	}

	sig, err := psa.identitySigner.Sign(message)
	if err != nil {
		return nil, err
	}
	return sig.Bytes(), nil
}

var _ payments_identity.Signer = paymentsSignerAdapter{}
