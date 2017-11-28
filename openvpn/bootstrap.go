package openvpn

import (
	"encoding/json"
	dto_openvpn "github.com/mysterium/node/openvpn/service_discovery/dto"
	dto_discovery "github.com/mysterium/node/service_discovery/dto"
)

func Bootstrap() {
	dto_discovery.RegisterServiceDefinitionUnserializer(
		"openvpn",
		func(rawDefinition *json.RawMessage) (dto_discovery.ServiceDefinition, error) {
			var definition dto_openvpn.ServiceDefinition
			err := json.Unmarshal(*rawDefinition, &definition)

			return definition, err
		},
	)

	dto_discovery.RegisterPaymentMethodUnserializer(
		dto_openvpn.PAYMENT_METHOD_PER_TIME,
		func(rawDefinition *json.RawMessage) (dto_discovery.PaymentMethod, error) {
			var method dto_openvpn.PaymentMethodPerTime
			err := json.Unmarshal(*rawDefinition, &method)

			return method, err
		},
	)

	dto_discovery.RegisterPaymentMethodUnserializer(
		dto_openvpn.PAYMENT_METHOD_PER_BYTES,
		func(rawDefinition *json.RawMessage) (dto_discovery.PaymentMethod, error) {
			var method dto_openvpn.PaymentMethodPerBytes
			err := json.Unmarshal(*rawDefinition, &method)

			return method, err
		},
	)
}
