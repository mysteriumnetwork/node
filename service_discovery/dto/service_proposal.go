package dto

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
}
