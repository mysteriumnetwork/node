package dto

type ServiceProposal struct {
	// A version number is included in the proposal to allow extensions to the proposal format
	Format string

	// Unique identifier of a provider
	ProviderId string

	// Per provider unique serial number of service description provided
	SerialNumber int

	// Qualitative service definition
	ServiceDefinition ServiceDefinition

	// Service payment & metering method
	PaymentMethod PaymentMethod
}
