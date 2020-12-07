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
	"errors"
	"strings"

	"github.com/mysteriumnetwork/metrics"
	"github.com/mysteriumnetwork/node/market"
)

var errEventNotImplemented = errors.New("event not implemented")

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
	if id, metric := mapEventToMetric(event); metric != nil {
		return transport.morqaClient.SendMetric(id, metric)
	}

	return errEventNotImplemented
}

func mapEventToMetric(event Event) (string, *metrics.Event) {
	switch event.EventName {
	case unlockEventName:
		return identityUnlockToMetricsEvent(event.Context.(string), event.Application)
	case sessionEventName:
		return sessionEventToMetricsEvent(event.Context.(sessionEventContext))
	case sessionDataName:
		return sessionDataToMetricsEvent(event.Context.(sessionDataContext))
	case sessionTokensName:
		return sessionTokensToMetricsEvent(event.Context.(sessionTokensContext))
	case proposalEventName:
		return proposalEventToMetricsEvent(event.Context.(market.ServiceProposal), event.Application)
	case traceEventName:
		return traceEventToMetricsEvent(event.Context.(sessionTraceContext), event.Application)
	case registerIdentity:
		return identityRegistrationEvent(event.Context.(registrationEvent), event.Application)
	case natMappingEventName:
		return natMappingEvent(event.Context.(natMappingContext), event.Application)
	}

	return "", nil
}

func natMappingEvent(context natMappingContext, info appInfo) (string, *metrics.Event) {
	var errMsg string
	if context.ErrorMessage != nil {
		errMsg = *context.ErrorMessage
	}

	return context.ID, &metrics.Event{
		IsProvider: true,
		Metric: &metrics.Event_NatMappingPayload{
			NatMappingPayload: &metrics.NatMappingPayload{
				Stage:      context.Stage,
				Successful: context.Successful,
				Err:        errMsg,
				Version: &metrics.VersionPayload{
					Version: info.Version,
					Os:      info.OS,
					Arch:    info.Arch,
				},
			},
		},
	}
}

func identityRegistrationEvent(data registrationEvent, info appInfo) (string, *metrics.Event) {
	return data.Identity, &metrics.Event{
		Metric: &metrics.Event_RegistrationPayload{
			RegistrationPayload: &metrics.RegistrationPayload{
				Version: &metrics.VersionPayload{
					Version: info.Version,
					Os:      info.OS,
					Arch:    info.Arch,
				},
				Country: data.Country,
				Status:  data.Status,
			},
		},
	}
}

func identityUnlockToMetricsEvent(id string, info appInfo) (string, *metrics.Event) {
	return id, &metrics.Event{
		Metric: &metrics.Event_VersionPayload{
			VersionPayload: &metrics.VersionPayload{
				Version: info.Version,
				Os:      info.OS,
				Arch:    info.Arch,
			},
		},
	}
}

func sessionEventToMetricsEvent(ctx sessionEventContext) (string, *metrics.Event) {
	return ctx.Consumer, &metrics.Event{
		TargetId:   ctx.Provider,
		IsProvider: false,
		Metric: &metrics.Event_SessionEventPayload{
			SessionEventPayload: &metrics.SessionEventPayload{
				Event: ctx.Event,
				Session: &metrics.SessionPayload{
					Id:              ctx.ID,
					ServiceType:     ctx.ServiceType,
					ProviderCountry: ctx.ProviderCountry,
					ConsumerCountry: ctx.ConsumerCountry,
					AccountantId:    ctx.AccountantID,
				},
			},
		},
	}
}

func sessionDataToMetricsEvent(ctx sessionDataContext) (string, *metrics.Event) {
	return ctx.Consumer, &metrics.Event{
		TargetId:   ctx.Provider,
		IsProvider: false,
		Metric: &metrics.Event_SessionStatisticsPayload{
			SessionStatisticsPayload: &metrics.SessionStatisticsPayload{
				BytesSent:     ctx.Tx,
				BytesReceived: ctx.Rx,
				Session: &metrics.SessionPayload{
					Id:              ctx.ID,
					ServiceType:     ctx.ServiceType,
					ProviderCountry: ctx.ProviderCountry,
					ConsumerCountry: ctx.ConsumerCountry,
					AccountantId:    ctx.AccountantID,
				},
			},
		},
	}
}

func sessionTokensToMetricsEvent(ctx sessionTokensContext) (string, *metrics.Event) {
	return ctx.Consumer, &metrics.Event{
		TargetId:   ctx.Provider,
		IsProvider: false,
		Metric: &metrics.Event_SessionTokensPayload{
			SessionTokensPayload: &metrics.SessionTokensPayload{
				Tokens: ctx.Tokens.Text(10),
				Session: &metrics.SessionPayload{
					Id:              ctx.ID,
					ServiceType:     ctx.ServiceType,
					ProviderCountry: ctx.ProviderCountry,
					ConsumerCountry: ctx.ConsumerCountry,
					AccountantId:    ctx.AccountantID,
				},
			},
		},
	}
}

func proposalEventToMetricsEvent(ctx market.ServiceProposal, info appInfo) (string, *metrics.Event) {
	location := ctx.ServiceDefinition.GetLocation()

	return ctx.ProviderID, &metrics.Event{
		IsProvider: true,
		Metric: &metrics.Event_ProposalPayload{
			ProposalPayload: &metrics.ProposalPayload{
				ProviderId:  ctx.ProviderID,
				ServiceType: ctx.ServiceType,
				NodeType:    location.NodeType,
				Country:     location.Country,
				Version: &metrics.VersionPayload{
					Version: info.Version,
					Os:      info.OS,
					Arch:    info.Arch,
				},
			},
		},
	}
}

func traceEventToMetricsEvent(ctx sessionTraceContext, info appInfo) (string, *metrics.Event) {
	sender, target, isProvider := ctx.Consumer, ctx.Provider, false
	// TODO Remove this workaround by generating&signing&publishing `metrics.Event` in same place
	if strings.HasPrefix(ctx.Stage, "Provider") {
		sender, target, isProvider = ctx.Provider, ctx.Consumer, true
	}

	return sender, &metrics.Event{
		TargetId:   target,
		IsProvider: isProvider,
		Metric: &metrics.Event_SessionTracePayload{
			SessionTracePayload: &metrics.SessionTracePayload{
				Duration: uint64(ctx.Duration.Nanoseconds()),
				Stage:    ctx.Stage,
				Session: &metrics.SessionPayload{
					Id:              ctx.ID,
					ServiceType:     ctx.ServiceType,
					ProviderCountry: ctx.ProviderCountry,
					ConsumerCountry: ctx.ConsumerCountry,
					AccountantId:    ctx.AccountantID,
				},
				Version: &metrics.VersionPayload{
					Version: info.Version,
					Os:      info.OS,
					Arch:    info.Arch,
				},
			},
		},
	}
}
