/*
 * Copyright (C) 2020 The "MysteriumNetwork/node" Authors.
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

package sleep

import (
	"sync"

	"github.com/mysteriumnetwork/node/eventbus"
)

// Event represents sleep event triggered by underlying OS
type Event int

const (
	// AppTopicSleepNotification represents sleep management Event notification
	AppTopicSleepNotification = "sleep_notification"

	// EventWakeup event sent to node after OS wakes up from sleep
	EventWakeup Event = iota
	// EventSleep event sent to node after OS goes to sleep
	EventSleep
)

var eventChannel chan Event

// Notifier represents sleep event notifier structure
type Notifier struct {
	eventbus eventbus.Publisher
	stop     chan struct{}
	stopOnce sync.Once
}

// NewNotifier create sleep events notifier
func NewNotifier(eventbus eventbus.Publisher) *Notifier {
	eventChannel = make(chan Event)
	return &Notifier{
		eventbus: eventbus,
		stop:     make(chan struct{}),
	}
}
