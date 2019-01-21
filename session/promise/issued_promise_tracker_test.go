package promise

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"

	"github.com/stretchr/testify/assert"

	"github.com/mysteriumnetwork/node/identity"

	"github.com/mysteriumnetwork/payments/promises"
)

var issuer = mockedIssuer{}
var consumer = identity.Identity{"0x1111111111111"}
var provider = identity.Identity{"0x2222222222222"}
var initialState = State{
	Seq:    1,
	Amount: 100,
}

func TestCurrentStatePromiseWithAddedAmountIsIssued(t *testing.T) {
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

func TestCurrentStateIsAlignedWithConsumer(t *testing.T) {
	tracker := NewConsumerTracker(initialState, consumer, provider, issuer)

	assert.NoError(t, tracker.AlignStateWithProvider(State{Seq: 1, Amount: 100}))

	p, err := tracker.IssuePromiseWithAddedAmount(100)
	assert.NoError(t, err)
	assert.Equal(t, int64(200), p.Promise.Amount)
	assert.Equal(t, int64(1), p.Promise.SeqNo)
}

func TestBiggerConsumerAmountIsRejected(t *testing.T) {
	tracker := NewConsumerTracker(initialState, consumer, provider, issuer)

	assert.Equal(t, UnexpectedAmount, tracker.AlignStateWithProvider(State{Seq: 1, Amount: 200}))
}

func TestSmallerConsumerAmountIsRejected(t *testing.T) {
	tracker := NewConsumerTracker(initialState, consumer, provider, issuer)

	assert.Equal(t, UnexpectedAmount, tracker.AlignStateWithProvider(State{Seq: 1, Amount: 0}))
}

func TestIncreasedSeqNumberIsAccepted(t *testing.T) {
	tracker := NewConsumerTracker(initialState, consumer, provider, issuer)

	assert.NoError(t, tracker.AlignStateWithProvider(State{Seq: 2, Amount: 0}))

	p, err := tracker.IssuePromiseWithAddedAmount(59)
	assert.NoError(t, err)
	assert.Equal(t, int64(59), p.Promise.Amount)
	assert.Equal(t, int64(2), p.Promise.SeqNo)
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
