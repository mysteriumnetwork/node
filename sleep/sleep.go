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
	"time"

	"github.com/rs/zerolog/log"
	"golang.org/x/net/context"

	"github.com/mysteriumnetwork/node/core/connection/connectionstate"
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
	eventBus          eventbus.EventBus
	stop              chan struct{}
	stopOnce          sync.Once
	connectionManager connectionManager
}

type connectionManager interface {
	// Status queries current status of connection
	Status(int) connectionstate.Status
	// CheckChannel checks if current session channel is alive, returns error on failed keep-alive ping
	CheckChannel(context.Context) error
	// Reconnect reconnects current session
	Reconnect(int)
}

// NewNotifier create sleep events notifier
func NewNotifier(manager connectionManager, eventbus eventbus.EventBus) *Notifier {
	eventChannel = make(chan Event)
	return &Notifier{
		connectionManager: manager,
		eventBus:          eventbus,
		stop:              make(chan struct{}),
	}
}

func (n *Notifier) handleSleepEvent(e Event) {
	switch e {
	case EventSleep:
		log.Info().Msg("Got sleep notification during live vpn session")
	case EventWakeup:
		log.Info().Msg("Got wake-up from sleep notification - checking if need to reconnect")
		if n.connectionManager.Status(0).State != connectionstate.Connected {
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
		defer cancel()
		if err := n.connectionManager.CheckChannel(ctx); err != nil {
			log.Info().Msgf("Channel dead - reconnecting: %s", err)
			n.connectionManager.Reconnect(0)
		} else {
			log.Info().Msg("Channel still alive - no need to reconnect")
		}
	}
}

// Subscribe subscribes to sleep notifications
func (n *Notifier) Subscribe() {
	n.eventBus.SubscribeAsync(AppTopicSleepNotification, n.handleSleepEvent)
}
