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
	"fmt"

	"github.com/mysteriumnetwork/metrics"
)

// NewMORQATransport creates transport allowing to send events to Mysterium Quality Oracle - MORQA
func NewMORQATransport(morqaClient *MysteriumMORQA) *morqaTransport {
	return &morqaTransport{
		morqaClient: morqaClient,
	}
}

type morqaTransport struct {
	morqaClient *MysteriumMORQA
}

func (transport *morqaTransport) SendEvent(event Event) error {
	if metric := mapEventToMetric(event); metric != nil {
		return transport.morqaClient.SendMetric(metric)
	}

	return fmt.Errorf("event not implemented")
}

func mapEventToMetric(event Event) *metrics.Event {
	switch event.EventName {
	case startupEventName:
		return nodeVersionToMetricsEvent(event.Application.Version)
	case sessionEventName:
		return sessionEventToMetricsEvent(event.Context.(sessionEventContext))
	case sessionDataName:
		return sessionDataToMetricsEvent(event.Context.(sessionDataContext))
	}
	return nil
}

func nodeVersionToMetricsEvent(version string) *metrics.Event {
	return &metrics.Event{
		Metric: &metrics.Event_VersionPayload{
			VersionPayload: &metrics.VersionPayload{
				Version: version,
			},
		},
	}
}

func sessionEventToMetricsEvent(context sessionEventContext) *metrics.Event {
	return &metrics.Event{
		Signature:  context.Consumer,
		TargetId:   context.Provider,
		IsProvider: false,
		Metric: &metrics.Event_SessionEventPayload{
			SessionEventPayload: &metrics.SessionEventPayload{
				Event: context.Event,
				Session: &metrics.SessionPayload{
					Id:             context.ID,
					ServiceType:    context.ServiceType,
					ProviderContry: context.ProviderCountry,
					ConsumerContry: context.ConsumerCountry,
				},
			},
		},
	}
}

func sessionDataToMetricsEvent(context sessionDataContext) *metrics.Event {
	return &metrics.Event{
		Signature:  context.Consumer,
		TargetId:   context.Provider,
		IsProvider: false,
		Metric: &metrics.Event_SessionStatisticsPayload{
			SessionStatisticsPayload: &metrics.SessionStatisticsPayload{
				BytesSent:     context.Tx,
				BytesReceived: context.Rx,
				Session: &metrics.SessionPayload{
					Id:             context.ID,
					ServiceType:    context.ServiceType,
					ProviderContry: context.ProviderCountry,
					ConsumerContry: context.ConsumerCountry,
				},
			},
		},
	}
}
