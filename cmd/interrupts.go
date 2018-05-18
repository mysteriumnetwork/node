/*
 * Copyright (C) 2018 The "MysteriumNetwork/node" Authors.
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
	"sync"
	"syscall"
)

// ApplicationStopper stops application and performs required cleanup tasks
type ApplicationStopper func()

// StopOnInterrupts invokes given stopper on SIGTERM and SIGHUP interrupts with additional wait condition
func StopOnInterruptsConditional(stop ApplicationStopper, stopWaiter *sync.WaitGroup) {
	sigterm := make(chan os.Signal)
	signal.Notify(sigterm, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)

	go waitTerminationSignalConditional(sigterm, stop, stopWaiter)
}

func waitTerminationSignalConditional(termination chan os.Signal, stop ApplicationStopper, stopWaiter *sync.WaitGroup) {
	stopWaiter.Wait()
	<-termination
	stop()
}

// StopOnInterrupts invokes given stopper on SIGTERM and SIGHUP interrupts
func StopOnInterrupts(stop ApplicationStopper) {
	sigterm := make(chan os.Signal)
	signal.Notify(sigterm, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)

	go waitTerminationSignal(sigterm, stop)
}

func waitTerminationSignal(termination chan os.Signal, stop ApplicationStopper) {
	<-termination
	stop()
}
