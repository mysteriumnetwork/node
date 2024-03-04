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

package eventbus

import (
	"fmt"
	"sync"

	asaskevichEventBus "github.com/mysteriumnetwork/EventBus"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// EventBus allows subscribing and publishing data by topic
type EventBus interface {
	Publisher
	Subscriber
}

// Publisher publishes events
type Publisher interface {
	Publish(topic string, data interface{})
}

// Subscriber subscribes to events.
type Subscriber interface {
	Subscribe(topic string, fn interface{}) error
	SubscribeAsync(topic string, fn interface{}) error
	Unsubscribe(topic string, fn interface{}) error
	UnsubscribeWithUID(topic, uid string, fn interface{}) error
	SubscribeWithUID(topic, uid string, fn interface{}) error
}

type simplifiedEventBus struct {
	bus asaskevichEventBus.Bus

	mu  sync.RWMutex
	sub map[string][]string
}

func (b *simplifiedEventBus) Unsubscribe(topic string, fn interface{}) error {
	return b.bus.Unsubscribe(topic, fn)
}

func (b *simplifiedEventBus) UnsubscribeWithUID(topic, uid string, fn interface{}) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if len(b.sub[topic]) == 0 {
		return fmt.Errorf("topic %s doesn't exist", topic)
	}

	for i, id := range b.sub[topic] {
		if id == uid {
			b.sub[topic] = append(b.sub[topic][:i], b.sub[topic][i+1:]...)

			break
		}
	}

	if len(b.sub[topic]) == 0 {
		delete(b.sub, topic)
	}

	return b.bus.Unsubscribe(topic+uid, fn)
}

func (b *simplifiedEventBus) Subscribe(topic string, fn interface{}) error {
	return b.bus.Subscribe(topic, fn)
}

func (b *simplifiedEventBus) SubscribeWithUID(topic, uid string, fn interface{}) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.sub[topic] = append(b.sub[topic], uid)

	return b.bus.Subscribe(topic+uid, fn)
}

func (b *simplifiedEventBus) SubscribeAsync(topic string, fn interface{}) error {
	return b.bus.SubscribeAsync(topic, fn, false)
}

func (b *simplifiedEventBus) Publish(topic string, data interface{}) {
	log.WithLevel(levelFor(topic)).Msgf("Published topic=%q event=%+v", topic, data)
	b.bus.Publish(topic, data)

	b.mu.RLock()
	ids := b.sub[topic]
	idsCopy := make([]string, len(ids))
	copy(idsCopy, ids)
	b.mu.RUnlock()

	for _, id := range idsCopy {
		b.bus.Publish(topic+id, data)
	}
}

// New returns implementation of EventBus.
func New() *simplifiedEventBus {
	return &simplifiedEventBus{
		bus: asaskevichEventBus.New(),
		sub: make(map[string][]string),
	}
}

var logLevelsByTopic = map[string]zerolog.Level{
	"ProposalAdded":            zerolog.Disabled,
	"ProposalUpdated":          zerolog.Disabled,
	"ProposalRemoved":          zerolog.Disabled,
	"proposalEvent":            zerolog.Disabled,
	"Statistics":               zerolog.Disabled,
	"Throughput":               zerolog.Disabled,
	"State change":             zerolog.TraceLevel,
	"Session data transferred": zerolog.TraceLevel,
	"Session change":           zerolog.TraceLevel,
	"hermes_promise_received":  zerolog.TraceLevel,
}

func levelFor(topic string) zerolog.Level {
	if level, exist := logLevelsByTopic[topic]; exist {
		return level
	}

	return zerolog.DebugLevel
}
