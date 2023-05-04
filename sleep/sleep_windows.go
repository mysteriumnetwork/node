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
	winlog "github.com/mysteriumnetwork/gowinlog"
	"github.com/rs/zerolog/log"
)

// Start starts event log notifier
func (n *Notifier) Start() {
	log.Debug().Msg("Register for sleep log events")
	watcher, err := winlog.NewWinLogWatcher()
	if err != nil {
		log.Error().Msgf("Couldn't create log watcher: %v\n", err)
		return
	}

	watcher.SubscribeFromNow("System", "*[System[Provider[@Name='Microsoft-Windows-Power-Troubleshooter'] and EventID=1]]")
	for {
		select {
		case <-watcher.Event():
			n.eventBus.Publish(AppTopicSleepNotification, EventWakeup)
		case err := <-watcher.Error():
			log.Error().Msgf("Log watcher error: %v\n", err)
		case <-n.stop:
			break
		}
	}
	watcher.Shutdown()
}

// Stop stops event log notifier
func (n *Notifier) Stop() {
	n.stopOnce.Do(func() {
		log.Debug().Msg("Unregister sleep log events watcher")
		close(n.stop)
	})
}
