package dto

type PaymentMethod interface {
	// // Service usage metering method
	GetType() string

	// Service price per unit of metering
	GetPrice() Price
}
