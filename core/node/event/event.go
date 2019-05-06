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

package event

const (
	// Topic represents the topic we're gonna be publishing and subscribing on
	Topic = "Node"
	// StatusStarted is published once node is started
	StatusStarted Status = "Started"
	// StatusStopped is published once node is stopped
	StatusStopped Status = "Stopped"
)

// Status represents the various states of node
type Status string

// Payload is the payload we'll send once an event is published
type Payload struct {
	Status Status
}
