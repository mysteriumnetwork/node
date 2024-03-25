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
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func Test_simplifiedEventBus_Publish_InvokesSubscribers(t *testing.T) {
	eventBus := New()
	var received string
	eventBus.Subscribe("test topic", func(data string) {
		received = data
	})

	eventBus.Publish("test topic", "test data")

	assert.Equal(t, "test data", received)
}

type handler struct {
	val int
}

func (h *handler) Handle(_ string) {
	h.val++
}

func TestUnsubscribeMethod(t *testing.T) {
	bus := New()
	h := &handler{val: 0}
	h2 := &handler{val: 5}

	bus.SubscribeWithUID("topic", "1", h.Handle)
	bus.SubscribeWithUID("topic", "2", h2.Handle)

	bus.Publish("topic", "1")

	err := bus.UnsubscribeWithUID("topic", "1", h.Handle)
	assert.NoError(t, err)

	bus.Publish("topic", "2")

	err = bus.UnsubscribeWithUID("topic", "2", h2.Handle)
	assert.NoError(t, err)

	err = bus.UnsubscribeWithUID("topic", "1", h.Handle)
	assert.Error(t, err)

	bus.Publish("topic", "3")

	assert.Equal(t, 1, h.val)
	assert.Equal(t, 7, h2.val)
}

func Test_simplifiedEventBus_Publish_DataRace(t *testing.T) {
	eventBus := New()

	fn := func(data string) {}

	active := new(atomic.Bool)
	active.Store(true)

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()

		for i := 0; i < 100; i++ {
			eventBus.SubscribeWithUID("topic", "1", fn)
			eventBus.Publish("topic", "test data")
			time.Sleep(time.Millisecond)
		}
		active.Store(false)
	}()
	go func() {
		defer wg.Done()
		for active.Load() == true {
			eventBus.UnsubscribeWithUID("topic", "1", fn)
			time.Sleep(time.Millisecond)
		}
	}()
	wg.Wait()
}
