package dto

type ServiceProposal struct {
	// Per provider unique serial number of service description provided
	Id int

	// A version number is included in the proposal to allow extensions to the proposal format
	Format string

	// Unique identifier of a provider
	ProviderId string

	// Definitions of service type offered
	ServiceType string

	// Qualitative service definition
	ServiceDefinition ServiceDefinition

	// Service payment & metering method
	PaymentMethod PaymentMethod
}
