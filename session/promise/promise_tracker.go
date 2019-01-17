package promise

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/payments/promises"
)

type Issuer interface {
	Issue(promise promises.Promise) (promises.IssuedPromise, error)
}

type PromiseState struct {
	seq    int64
	amount int64
}

type ConsumerTracker struct {
	currentState PromiseState
	consumer     identity.Identity
	receiver     identity.Identity
	issuer       Issuer
}

func NewConsumerTracker(initial PromiseState, consumer, provider identity.Identity, issuer Issuer) *ConsumerTracker {
	return &ConsumerTracker{
		currentState: initial,
		consumer:     consumer,
		receiver:     provider,
		issuer:       issuer,
	}
}

func (t *ConsumerTracker) AlignStateWithProvider(providerState PromiseState) error {
	return nil
}

func (t *ConsumerTracker) IssuePromiseWithAddedAmount(amountToAdd int64) (promises.IssuedPromise, error) {

	promise := promises.Promise{
		Extra: ExtraData{
			ConsumerAddress: common.HexToAddress(t.consumer.Address),
		},
		Receiver: common.HexToAddress(t.receiver.Address),
		Amount:   t.currentState.amount + amountToAdd,
		SeqNo:    t.currentState.seq,
	}
	return t.issuer.Issue(promise)
}
