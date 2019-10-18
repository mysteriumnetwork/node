/*
 * Copyright (C) 2019 The "MysteriumNetwork/node" Authors.
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */

package transactor

import (
	"sync"
	"time"

	"github.com/mysteriumnetwork/node/core/discovery"
	"github.com/mysteriumnetwork/node/identity"
)

type transactorCaller interface {
	TopUp(id string) error
}

// DefaultRetryCount represents the default max retries we'll do before abandoning a top up attempt
const DefaultRetryCount = 3

// DefaultDelayBetweenRetries represents the delay between retries
var DefaultDelayBetweenRetries = time.Minute * 1

type eventBus interface {
	SubscribeAsync(topic string, fn interface{}) error
}

// TopUpper tops up accounts on relevant identity related events
type TopUpper struct {
	tc                transactorCaller
	retryAttempts     int
	delayBetweenRetry time.Duration
	once              sync.Once
	stop              chan struct{}
}

// NewTopUpper returns a new instance of topper upper
func NewTopUpper(transactor transactorCaller, retries int, delay time.Duration) *TopUpper {
	return &TopUpper{
		tc:                transactor,
		retryAttempts:     retries,
		delayBetweenRetry: delay,
		stop:              make(chan struct{}),
	}
}

// Subscribe subscribes to relevant events
func (tu *TopUpper) Subscribe(eb eventBus) error {
	return eb.SubscribeAsync(discovery.IdentityRegistrationTopic, tu.handleRegistrationEvent)
}

// Stop stops the topper upper
func (tu *TopUpper) Stop() {
	tu.once.Do(func() {
		close(tu.stop)
	})
}

func (tu *TopUpper) handleRegistrationEvent(id identity.Identity) {
	for i := 0; i < tu.retryAttempts; i++ {
		err := tu.tc.TopUp(id.Address)
		if err != nil {
			log.Warnf("could not top up newly registered identity channel %v. Will retry", err)
			if i+1 == tu.retryAttempts {
				log.Errorf("top up failed after multiple attempts, aborting", err)
			}
		}

		// Block until we are either stopped or time has come for a retry
		select {
		case <-tu.stop:
			return
		case <-time.After(tu.delayBetweenRetry):
		}
	}
}
