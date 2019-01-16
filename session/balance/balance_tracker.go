package balance

import (
	"time"

	log "github.com/cihub/seelog"
	"github.com/mysteriumnetwork/node/money"
)

const balanceTrackerPrefix = "[balance-tracker] "

type Sender interface {
	Send(BalanceMessage) error
}

type TimeKeeper interface {
	StartTracking()
	Elapsed() time.Duration
}

type AmountCalculator interface {
	TotalAmount(duration time.Duration) money.Money
}

type ProviderBalanceTracker struct {
	sender           Sender
	timeKeeper       TimeKeeper
	amountCalculator AmountCalculator
	period           time.Duration

	totalPromised uint64
	balance       uint64
	stop          chan struct{}
}

func NewProviderBalanceTracker(sender Sender, timeKeeper TimeKeeper, amountCalculator AmountCalculator, period time.Duration, initialBalance uint64) *ProviderBalanceTracker {
	return &ProviderBalanceTracker{
		sender:           sender,
		timeKeeper:       timeKeeper,
		period:           period,
		amountCalculator: amountCalculator,
		totalPromised:    initialBalance,

		stop: make(chan struct{}),
	}
}

func (pbt *ProviderBalanceTracker) calculateBalance() {
	cost := pbt.amountCalculator.TotalAmount(pbt.timeKeeper.Elapsed())
	pbt.balance = pbt.totalPromised - cost.Amount
}

func (pbt *ProviderBalanceTracker) periodicSend() error {
	for {
		select {
		case <-pbt.stop:
			return nil
		case <-time.After(pbt.period):
			pbt.calculateBalance()
			// TODO: Maybe retry a couple of times?
			err := pbt.sendMessage()
			if err != nil {
				log.Error(balanceTrackerPrefix, "Balance tracker failed to send the balance message")
				log.Error(balanceTrackerPrefix, err)
			}
			// TODO: destroy session/connection if balance negative? or should we bubble the error and let the caller be responsible for this?
			// TODO: wait for response here on the promise topic
		}
	}
}

func (pbt *ProviderBalanceTracker) sendMessage() error {
	return pbt.sender.Send(pbt.getBalanceMessage())
}

func (pbt *ProviderBalanceTracker) getBalanceMessage() BalanceMessage {
	// TODO: sequence ID should come here, somehow
	return BalanceMessage{
		SequenceID: 0,
		Balance:    pbt.balance,
	}
}

func (pbt *ProviderBalanceTracker) Track() error {
	pbt.timeKeeper.StartTracking()
	return pbt.periodicSend()
}

func (pbt *ProviderBalanceTracker) Stop() {
	pbt.stop <- struct{}{}
}
