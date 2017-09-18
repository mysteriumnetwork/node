package dto

type PaymentMethod interface {
	// Service price per unit of metering
	GetPrice() Money
}
