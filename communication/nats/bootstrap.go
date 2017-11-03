package nats

import (
	"encoding/json"
	dto_discovery "github.com/mysterium/node/service_discovery/dto"
)

func Bootstrap() {
	dto_discovery.RegisterContactDefinitionUnserializer(
		CONTACT_NATS_V1,
		func(rawDefinition *json.RawMessage) (dto_discovery.ContactDefinition, error) {
			var contact ContactNATSV1

			err := json.Unmarshal(*rawDefinition, &contact)

			return contact, err
		},
	)
}
