package noop

import (
	"encoding/json"

	dto_discovery "github.com/mysteriumnetwork/node/service_discovery/dto"
)

// Bootstrap is called on program initialization time and registers various deserializers related to opepnvpn service
func Bootstrap() {
	dto_discovery.RegisterServiceDefinitionUnserializer(
		ServiceType,
		func(rawDefinition *json.RawMessage) (dto_discovery.ServiceDefinition, error) {
			var definition ServiceDefinition
			err := json.Unmarshal(*rawDefinition, &definition)

			return definition, err
		},
	)

	dto_discovery.RegisterPaymentMethodUnserializer(
		PaymentMethodNoop,
		func(rawDefinition *json.RawMessage) (dto_discovery.PaymentMethod, error) {
			var method PaymentNoop
			err := json.Unmarshal(*rawDefinition, &method)

			return method, err
		},
	)
}
