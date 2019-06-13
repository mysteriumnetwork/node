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

package quality

import (
	"github.com/mysteriumnetwork/metrics"
)

var (
	emptyMetric = &metrics.Event{}
)

// NewMORQATransport creates transport allowing to send events to Mysterium Quality Oracle - MORQA
func NewMORQATransport(morqaClient *MysteriumMORQA, backupTransport Transport) *morqaTransport {
	return &morqaTransport{
		morqaClient:     morqaClient,
		backupTransport: backupTransport,
	}
}

type morqaTransport struct {
	morqaClient     *MysteriumMORQA
	backupTransport Transport
}

func (transport *morqaTransport) SendEvent(event Event) error {
	if metric := mapEventToMetric(event); metric != emptyMetric {
		err := transport.morqaClient.SendMetric(metric)
		if err != nil {
			return err
		}
	}

	return transport.backupTransport.SendEvent(event)
}

func mapEventToMetric(event Event) *metrics.Event {
	switch event.EventName {
	case startupEventName:
		return &metrics.Event{
			IsProvider: false,
			TargetId:   "0x1",
			Metric: &metrics.Event_VersionPayload{
				VersionPayload: &metrics.VersionPayload{
					Version: event.Application.Version,
				},
			},
		}
	}
	return emptyMetric
}
