/*
 * Copyright (C) 2017 The "MysteriumNetwork/node" Authors.
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

package cmd

import (
	"os"
	"os/signal"
	"syscall"
)

// SignalCallback is invoked when process receives signals defined below
type SignalCallback func()

// RegisterSignalCallback registers given callback to call on SIGTERM and SIGHUP interrupts
func RegisterSignalCallback(callback SignalCallback) {
	sigterm := make(chan os.Signal, 1)
	signal.Notify(sigterm, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)

	go waitTerminationSignal(sigterm, callback)
}

func waitTerminationSignal(termination chan os.Signal, callback SignalCallback) {
	<-termination
	callback()
}
