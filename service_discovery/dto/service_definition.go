package dto

type ServiceDefinition interface {
	GetType() string
	GetLocation() Location
}
