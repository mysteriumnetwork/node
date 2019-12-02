package mmn

import (
	"github.com/mysteriumnetwork/node/core/connection"
	"github.com/mysteriumnetwork/node/core/service"
	"github.com/mysteriumnetwork/node/eventbus"
	"github.com/mysteriumnetwork/node/identity"

	"github.com/rs/zerolog/log"
)

type MMN struct {
	collector *Collector
	client    *client
	eventBus  eventbus.EventBus
}

func NewMMN(collector *Collector, client *client, eventBus eventbus.EventBus) *MMN {
	return &MMN{collector, client, eventBus}
}

func (m *MMN) SubscribeToEvents() {
	m.subscribeToRegistrationEvent()
	m.subscribeToProviderEvent()
	m.subscribeToConsumerEvent()
}

func (m *MMN) Register(identity string) error {
	m.collector.SetIdentity(identity)

	return m.client.RegisterNode(m.collector.GetCollectedInformation())
}

func (m *MMN) UpdateNodeType(isProvider bool) error {
	// don't resend when node type is the same
	if m.collector.GetNodeType() == isProvider {
		return nil
	}

	m.collector.SetNodeType(isProvider)

	return m.client.UpdateNodeType(m.collector.GetCollectedInformation())
}

func (m *MMN) subscribeToRegistrationEvent() {
	m.eventBus.SubscribeAsync(
		identity.IdentityUnlockTopic,
		func(identity string) {
			log.Debug().Msg("Registration event")

			if err := m.Register(identity); err != nil {
				log.Error().Msg("Failed to register to MMN: " + err.Error())
			}
		},
	)
}

func (m *MMN) subscribeToConsumerEvent() {
	m.subscribeToNodeTypeEvent(connection.SessionEventTopic, false)
}

func (m *MMN) subscribeToProviderEvent() {
	m.subscribeToNodeTypeEvent(service.StatusTopic, true)
}

func (m *MMN) subscribeToNodeTypeEvent(event string, isProvider bool) {
	m.eventBus.SubscribeAsync(
		event,
		func(i interface{}) {
			log.Debug().Msgf("Node type (provider: %v) event", isProvider)

			if err := m.UpdateNodeType(isProvider); err != nil {
				log.Error().Msg("Failed to send node type to MMN: " + err.Error())
			}
		},
	)
}
