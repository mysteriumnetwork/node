package dto

import (
	"encoding/json"
	"errors"
)

type ServiceProposal struct {
	// Per provider unique serial number of service description provided
	Id int `json:"id"`

	// A version number is included in the proposal to allow extensions to the proposal format
	Format string `json:"format"`

	// Type of service type offered
	ServiceType string `json:"service_type"`

	// Qualitative service definition
	ServiceDefinition ServiceDefinition `json:"service_definition"`

	// Type of service payment method
	PaymentMethodType string `json:"payment_method_type"`

	// Service payment & usage metering definition
	PaymentMethod PaymentMethod `json:"payment_method"`

	// Unique identifier of a provider
	ProviderId Identity `json:"provider_id"`

	// Communication methods possible
	ProviderContacts []Contact `json:"provider_contacts"`

	// Connection string
	ConnectionConfig string `json:"connection_string,omitempty"`
}

type ServiceDefinitionUnserializer func(*json.RawMessage) (ServiceDefinition, error)
// service definition unserializer registry
var serviceDefinitionMap map[string]ServiceDefinitionUnserializer = make(map[string]ServiceDefinitionUnserializer, 10)

func RegisterServiceDefinitionUnserializer(serviceType string, unserializer ServiceDefinitionUnserializer) {
	serviceDefinitionMap[serviceType] = unserializer
}
func unserializeServiceDefinition(serviceType string, message *json.RawMessage) (ServiceDefinition, error) {
	if method, ok := serviceDefinitionMap[serviceType]; ok {
		return method(message)
	}

	return nil, errors.New("Service unserializer '" + serviceType + "' doesn't exist")
}

type PaymentMethodUnserializer func(*json.RawMessage) (PaymentMethod, error)
// service payment method unserializer registry
var paymentMethodMap map[string]PaymentMethodUnserializer = make(map[string]PaymentMethodUnserializer, 0)

func RegisterPaymentMethodUnserializer(paymentMethod string, unserializer func(*json.RawMessage) (PaymentMethod, error)) {
	paymentMethodMap[paymentMethod] = unserializer
}

func unserializePaymentMethod(paymentMethod string, message *json.RawMessage) (PaymentMethod, error) {
	if method, ok := paymentMethodMap[paymentMethod]; ok {
		return method(message)
	}

	return nil, errors.New("Payment method unserializer '" + paymentMethod + "' doesn't exist")
}

func (genericProposal *ServiceProposal) UnmarshalJSON(data []byte) (err error) {
	var jsonData struct {
		Id                int              `json:"id"`
		Format            string           `json:"format"`
		ServiceType       string           `json:"service_type"`
		ProviderId        string           `json:"provider_id"`
		PaymentMethodType string           `json:"payment_method_type"`
		ServiceDefinition *json.RawMessage `json:"service_definition"`
		PaymentMethod     *json.RawMessage `json:"payment_method"`
		ProviderContacts  []Contact        `json:"provider_contacts"`
	}
	if err = json.Unmarshal(data, &jsonData); err != nil {
		return
	}

	genericProposal.Id = jsonData.Id
	genericProposal.Format = jsonData.Format
	genericProposal.ServiceType = jsonData.ServiceType
	genericProposal.ProviderId = Identity(jsonData.ProviderId)
	genericProposal.PaymentMethodType = jsonData.PaymentMethodType
	genericProposal.ProviderContacts = jsonData.ProviderContacts

	// run the service definition implementation from our registry
	genericProposal.ServiceDefinition, err = unserializeServiceDefinition(
		jsonData.ServiceType,
		jsonData.ServiceDefinition,
	)

	if err != nil {
		return
	}

	// run the payment method implementation from our registry
	genericProposal.PaymentMethod, err = unserializePaymentMethod(
		jsonData.PaymentMethodType,
		jsonData.PaymentMethod,
	)

	if err != nil {
		return
	}

	return nil
}
