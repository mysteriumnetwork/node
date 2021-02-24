// +build !packageIOS

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

// #cgo LDFLAGS: -framework CoreFoundation -framework IOKit
// void NotifyWake();
// void NotifySleep();
// #include "darwin.h"
import "C"

import (
	"github.com/rs/zerolog/log"
)

// Start starts events notifier
func (n *Notifier) Start() {
	log.Debug().Msg("Register for sleep events")

	go C.registerNotifications()

	for {
		select {
		case e := <-eventChannel:
			n.eventBus.Publish(AppTopicSleepNotification, e)
		case <-n.stop:
			break
		}
	}
}

// Stop stops events notifier
func (n *Notifier) Stop() {
	n.stopOnce.Do(func() {
		log.Debug().Msg("Unregister sleep events")
		C.unregisterNotifications()
		close(n.stop)
	})
}
