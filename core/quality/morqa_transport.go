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
	"time"

	"github.com/mysteriumnetwork/metrics"
	"github.com/mysteriumnetwork/node/config"
	"github.com/mysteriumnetwork/node/core/location/locationstate"
	"github.com/mysteriumnetwork/node/market"
)

var errEventNotImplemented = errors.New("event not implemented")

type locationProvider interface {
	GetOrigin() locationstate.Location
}

// NewMORQATransport creates transport allowing to send events to Mysterium Quality Oracle - MORQA.
func NewMORQATransport(morqaClient *MysteriumMORQA, lp locationProvider) *morqaTransport {
	return &morqaTransport{
		morqaClient: morqaClient,
		lp:          lp,
	}
}

type morqaTransport struct {
	morqaClient *MysteriumMORQA
	lp          locationProvider
}

func (t *morqaTransport) SendEvent(event Event) error {
	if id, metric := mapEventToMetric(event); metric != nil {
		metric.Version = &metrics.VersionPayload{
			Version:         event.Application.Version,
			Os:              event.Application.OS,
			Arch:            event.Application.Arch,
			LauncherVersion: event.Application.LauncherVersion,
			HostOs:          event.Application.HostOS,
		}
		metric.Country = t.lp.GetOrigin().Country
		return t.morqaClient.SendMetric(id, metric)
	}

	return errEventNotImplemented
}

func mapEventToMetric(event Event) (string, *metrics.Event) {
	switch event.EventName {
	case pingEventName:
		return pingEventToMetricsEvent(event.Context.(pingEventContext))
	case unlockEventName:
		return identityUnlockToMetricsEvent(event.Context.(string))
	case sessionEventName:
		return sessionEventToMetricsEvent(event.Context.(sessionEventContext))
	case sessionDataName:
		return sessionDataToMetricsEvent(event.Context.(sessionDataContext))
	case sessionTokensName:
		return sessionTokensToMetricsEvent(event.Context.(sessionTokensContext))
	case proposalEventName:
		return proposalEventToMetricsEvent(event.Context.(market.ServiceProposal))
	case traceEventName:
		return traceEventToMetricsEvent(event.Context.(sessionTraceContext))
	case registerIdentity:
		return identityRegistrationEvent(event.Context.(registrationEvent))
	case natMappingEventName:
		return natMappingEvent(event.Context.(natMappingContext))
	case connectionEvent:
		return connectionEventToMetricsEvent(event.Context.(ConnectionEvent))
	case residentCountryEventName:
		return residentCountryToMetricsEvent(event.Context.(residentCountryEvent))
	case stunDetectionEvent:
		return natTypeToMetricsEvent(event.Context.(natTypeEvent))
	case natTypeDetectionEvent:
		return natTypeToMetricsEvent(event.Context.(natTypeEvent))
	case natTraversalMethod:
		return natTraversalMethodToMetricsEvent(event.Context.(natMethodEvent))
	}

	return "", nil
}

func natTraversalMethodToMetricsEvent(event natMethodEvent) (string, *metrics.Event) {
	return event.ID, &metrics.Event{
		IsProvider: true,
		Metric: &metrics.Event_NatMethod{
			NatMethod: &metrics.NatMethodResult{
				Method:  event.NATMethod,
				Success: event.Success,
			},
		},
	}
}

func natTypeToMetricsEvent(event natTypeEvent) (string, *metrics.Event) {
	return event.ID, &metrics.Event{
		Metric: &metrics.Event_StunDetection{
			StunDetection: &metrics.STUNDetectionPayload{
				Type: event.NATType,
			},
		},
	}
}

func residentCountryToMetricsEvent(event residentCountryEvent) (string, *metrics.Event) {
	return event.ID, &metrics.Event{
		Metric: &metrics.Event_ResidentCountry{
			ResidentCountry: &metrics.ResidentCountryPayload{
				Country: event.Country,
			},
		},
	}
}

func pingEventToMetricsEvent(context pingEventContext) (string, *metrics.Event) {
	sender, target, country := context.Consumer, context.Provider, context.ProviderCountry
	if context.IsProvider {
		sender, target, country = context.Provider, context.Consumer, context.ConsumerCountry
	}

	return sender, &metrics.Event{
		IsProvider: context.IsProvider,
		TargetId:   target,
		Metric: &metrics.Event_PingEvent{
			PingEvent: &metrics.PingPayload{
				SessionId:     context.ID,
				RemoteCountry: country,
				Duration:      uint64(context.Duration),
			},
		},
	}
}

func connectionEventToMetricsEvent(context ConnectionEvent) (string, *metrics.Event) {
	return context.ConsumerID, &metrics.Event{
		IsProvider: false,
		TargetId:   context.ProviderID,
		Metric: &metrics.Event_ConnectionEvent{
			ConnectionEvent: &metrics.ConnectionEvent{
				ServiceType: context.ServiceType,
				HermesId:    context.HermesID,
				Stage:       context.Stage,
				Error:       context.Error,
			},
		},
	}
}

func natMappingEvent(context natMappingContext) (string, *metrics.Event) {
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
			},
		},
	}
}

func identityRegistrationEvent(data registrationEvent) (string, *metrics.Event) {
	return data.Identity, &metrics.Event{
		Metric: &metrics.Event_RegistrationPayload{
			RegistrationPayload: &metrics.RegistrationPayload{
				Status: data.Status,
			},
		},
	}
}

func identityUnlockToMetricsEvent(id string) (string, *metrics.Event) {
	return id, &metrics.Event{}
}

func sessionEventToMetricsEvent(ctx sessionEventContext) (string, *metrics.Event) {
	sender, target, country := ctx.Consumer, ctx.Provider, ctx.ProviderCountry
	if ctx.IsProvider {
		sender, target, country = ctx.Provider, ctx.Consumer, ctx.ConsumerCountry
	}

	return sender, &metrics.Event{
		TargetId:   target,
		IsProvider: ctx.IsProvider,
		Metric: &metrics.Event_SessionEventPayload{
			SessionEventPayload: &metrics.SessionEventPayload{
				Event: ctx.Event,
				Session: &metrics.SessionPayload{
					Id:            ctx.ID,
					ServiceType:   ctx.ServiceType,
					RemoteCountry: country,
					HermesId:      ctx.AccountantID,
				},
			},
		},
	}
}

func sessionDataToMetricsEvent(ctx sessionDataContext) (string, *metrics.Event) {
	sender, target, country := ctx.Consumer, ctx.Provider, ctx.ProviderCountry
	if ctx.IsProvider {
		sender, target, country = ctx.Provider, ctx.Consumer, ctx.ConsumerCountry
	}

	return sender, &metrics.Event{
		TargetId:   target,
		IsProvider: ctx.IsProvider,
		Metric: &metrics.Event_SessionStatisticsPayload{
			SessionStatisticsPayload: &metrics.SessionStatisticsPayload{
				BytesSent:     ctx.Tx,
				BytesReceived: ctx.Rx,
				Duration:      duration(ctx.StartedAt),
				Session: &metrics.SessionPayload{
					Id:            ctx.ID,
					ServiceType:   ctx.ServiceType,
					RemoteCountry: country,
					HermesId:      ctx.AccountantID,
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
					Id:            ctx.ID,
					ServiceType:   ctx.ServiceType,
					RemoteCountry: ctx.ProviderCountry,
					HermesId:      ctx.AccountantID,
				},
			},
		},
	}
}

func proposalEventToMetricsEvent(ctx market.ServiceProposal) (string, *metrics.Event) {
	return ctx.ProviderID, &metrics.Event{
		IsProvider: true,
		Metric: &metrics.Event_ProposalPayload{
			ProposalPayload: &metrics.ProposalPayload{
				ServiceType: ctx.ServiceType,
				NodeType:    ctx.Location.IPType,
				VendorId:    config.GetString(config.FlagVendorID),
			},
		},
	}
}

func traceEventToMetricsEvent(ctx sessionTraceContext) (string, *metrics.Event) {
	sender, target, isProvider, country := ctx.Consumer, ctx.Provider, false, ctx.ProviderCountry
	// TODO Remove this workaround by generating&signing&publishing `metrics.Event` in same place
	if strings.HasPrefix(ctx.Stage, "Provider") {
		sender, target, isProvider, country = ctx.Provider, ctx.Consumer, true, ctx.ConsumerCountry
	}

	return sender, &metrics.Event{
		TargetId:   target,
		IsProvider: isProvider,
		Metric: &metrics.Event_SessionTracePayload{
			SessionTracePayload: &metrics.SessionTracePayload{
				Duration: uint64(ctx.Duration.Nanoseconds()),
				Stage:    ctx.Stage,
				Session: &metrics.SessionPayload{
					Id:            ctx.ID,
					ServiceType:   ctx.ServiceType,
					RemoteCountry: country,
					HermesId:      ctx.AccountantID,
				},
			},
		},
	}
}

func duration(startedAt time.Time) uint64 {
	if startedAt.IsZero() {
		return 0
	}

	return uint64(time.Since(startedAt).Seconds())
}
