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

package bus

import asaskevichEventBus "github.com/asaskevich/EventBus"

// EventBus allows subscribing and publishing data by topic
type EventBus interface {
	Subscribe(topic string, fn interface{}) error
	Publish(topic string, data interface{})
}

type simplifiedEventBus struct {
	bus asaskevichEventBus.Bus
}

func (bus simplifiedEventBus) Subscribe(topic string, fn interface{}) error {
	return bus.bus.Subscribe(topic, fn)
}

func (bus simplifiedEventBus) Publish(topic string, data interface{}) {
	bus.bus.Publish(topic, data)
}

// NewEventBus returns implementation of EventBus
func NewEventBus() EventBus {
	bus := asaskevichEventBus.New()
	return simplifiedEventBus{bus: bus}
}
