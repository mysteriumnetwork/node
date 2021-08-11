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

// AppTopicTraversal the topic that traversal events are published on
const AppTopicTraversal = "Traversal"

// BuildSuccessfulEvent returns new event for successful NAT traversal
func BuildSuccessfulEvent(id, stage string) Event {
	return Event{ID: id, Stage: stage, Successful: true}
}

// BuildFailureEvent returns new event for failed NAT traversal
func BuildFailureEvent(id, stage string, err error) Event {
	return Event{ID: id, Stage: stage, Successful: false, Error: err}
}

// Event represents a NAT traversal related event
type Event struct {
	ID         string `json:"id"`
	Stage      string `json:"stage"`
	Successful bool   `json:"successful"`
	Error      error  `json:"error,omitempty"`
}
