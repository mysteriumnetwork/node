package dto

import (
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"testing"
	"github.com/mysterium/node/money"
	"time"
)

func TestPaymentMethodPerTimeSerialize(t *testing.T) {
	price := money.NewMoney(0.5, money.CURRENCY_MYST)

	var tests = []struct {
		model        PaymentMethodPerTime
		expectedJson string
	}{
		{
			PaymentMethodPerTime{
				Price:    price,
				Duration: time.Duration(10),
			},
			`{
				"price": {
					"amount": 50000000,
					"currency": "MYST"
				},
				"duration": 10
			}`,
		},
		{
			PaymentMethodPerTime{},
			`{
				"price": {
					"amount": 0,
					"currency": ""
				},
				"duration": 0
			}`,
		},
	}

	for _, test := range tests {
		jsonBytes, err := json.Marshal(test.model)

		assert.Nil(t, err)
		assert.JSONEq(t, test.expectedJson, string(jsonBytes))
	}
}

func TestPaymentMethodPerTimeUnserialize(t *testing.T) {
	price := money.NewMoney(0.5, money.CURRENCY_MYST)

	var tests = []struct {
		json          string
		expectedModel PaymentMethodPerTime
		expectedError error
	}{
		{
			`{
				"price": {
					"amount": 50000000,
					"currency": "MYST"
				},
				"duration": 10
			}`,
			PaymentMethodPerTime{
				Price:    price,
				Duration: time.Duration(10),
			},
			nil,
		},
		{
			`{
				"price": {
					"amount": 0,
					"currency": ""
				},
				"duration": 0
			}`,
			PaymentMethodPerTime{},
			nil,
		},
		{
			`{}`,
			PaymentMethodPerTime{},
			nil,
		},
	}

	for _, test := range tests {
		var model PaymentMethodPerTime
		err := json.Unmarshal([]byte(test.json), &model)

		assert.Equal(t, test.expectedModel, model)
		if test.expectedError != nil {
			assert.EqualError(t, err, test.expectedError.Error())
		} else {
			assert.NoError(t, err)
		}
	}
}
