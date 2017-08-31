package dto

type ServiceProposal struct {
	// Per provider unique serial number of service description provided
	Id int

	// A version number is included in the proposal to allow extensions to the proposal format
	Format string

	// Type of service type offered
	ServiceType string

	// Qualitative service definition
	ServiceDefinition ServiceDefinition

	// Type of service payment method
	PaymentMethodType string

	// Service payment & usage metering definition
	PaymentMethod PaymentMethod

	// Unique identifier of a provider
	ProviderId string

	// Communication methods possible
	ProviderContacts []Contact
}
