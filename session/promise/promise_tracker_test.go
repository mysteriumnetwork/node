package promise

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"

	"github.com/stretchr/testify/assert"

	"github.com/mysteriumnetwork/node/identity"

	"github.com/mysteriumnetwork/payments/promises"
)

func TestCurrentStatePromiseWithAddedAmountIsIssued(t *testing.T) {
	issuer := mockedIssuer{}
	consumer := identity.Identity{"0x1111111111111"}
	provider := identity.Identity{"0x2222222222222"}
	initialState := PromiseState{
		seq:    1,
		amount: 100,
	}

	tracker := NewConsumerTracker(initialState, consumer, provider, issuer)
	p, err := tracker.IssuePromiseWithAddedAmount(200)
	assert.NoError(t, err)
	assert.Equal(
		t,
		promises.Promise{
			Receiver: common.HexToAddress(provider.Address),
			Extra: ExtraData{
				ConsumerAddress: common.HexToAddress(consumer.Address),
			},
			SeqNo:  1,
			Amount: 300,
		},
		p.Promise,
	)
}

type mockedIssuer struct {
}

func (issuer mockedIssuer) Issue(promise promises.Promise) (promises.IssuedPromise, error) {
	return promises.IssuedPromise{
		Promise:         promise,
		IssuerSignature: []byte("0xdeadbeef"),
	}, nil
}

var _ Issuer = mockedIssuer{}
