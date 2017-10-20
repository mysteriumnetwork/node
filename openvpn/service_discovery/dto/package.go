package dto

import (
	dto_discovery "github.com/mysterium/node/service_discovery/dto"
	"encoding/json"
	"github.com/mysterium/node/communication/nats"
)

func Initialize() {

}

func init() {
	dto_discovery.RegisterServiceDefinitionUnserializer(
		"openvpn",
		func(rawDefinition *json.RawMessage) (dto_discovery.ServiceDefinition, error) {
			var definition ServiceDefinition
			err := json.Unmarshal(*rawDefinition, &definition)

			return definition, err
 		},
	)

	dto_discovery.RegisterPaymentMethodUnserializer(
		PAYMENT_METHOD_PER_TIME,
		func(rawDefinition *json.RawMessage) (dto_discovery.PaymentMethod, error) {
			var method PaymentMethodPerTime
			err := json.Unmarshal(*rawDefinition, &method)

			return method, err
		},
	)

	dto_discovery.RegisterPaymentMethodUnserializer(
		PAYMENT_METHOD_PER_BYTES,
		func(rawDefinition *json.RawMessage) (dto_discovery.PaymentMethod, error) {
			var method PaymentMethodPerBytes
			err := json.Unmarshal(*rawDefinition, &method)

			return method, err
		},
	)

	dto_discovery.RegisterContactDefinitionUnserializer(
		nats.CONTACT_NATS_V1,
		func (rawDefinition *json.RawMessage) (dto_discovery.ContactDefinition, error) {
			var contact nats.ContactNATSV1

			err := json.Unmarshal(*rawDefinition, &contact)

			return contact, err
		},
	)
}
