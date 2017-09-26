package dto

import "github.com/mysterium/node/service_discovery/dto"

type PromiseBody struct {
	SerialNumber int
	IssuerId     string
	BenefiterId  string
	Amount       dto.Money
}