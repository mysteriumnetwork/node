package dto

import (
	"encoding/json"
	"github.com/mysterium/node/datasize"
	"github.com/mysterium/node/money"
	"github.com/stretchr/testify/assert"
	"testing"
)

var (
	price = money.NewMoney(0.5, money.CURRENCY_MYST)
)

func TestPaymentMethodPerBytesSerialize(t *testing.T) {
	var tests = []struct {
		model        PaymentMethodPerBytes
		expectedJson string
	}{
		{
			PaymentMethodPerBytes{
				Price: price,
				Bytes: datasize.Gigabyte,
			},
			`{
				"price": {
					"amount": 50000000,
					"currency": "MYST"
				},
				"bytes": 8589934592
			}`,
		},
		{
			PaymentMethodPerBytes{},
			`{
				"price": {}
			}`,
		},
	}

	for _, test := range tests {
		jsonBytes, err := json.Marshal(test.model)

		assert.Nil(t, err)
		assert.JSONEq(t, test.expectedJson, string(jsonBytes))
	}
}

func TestPaymentMethodPerBytesUnserialize(t *testing.T) {
	var tests = []struct {
		json          string
		expectedModel PaymentMethodPerBytes
		expectedError error
	}{
		{
			`{
				"price": {
					"amount": 50000000,
					"currency": "MYST"
				},
				"bytes": 8589934592
			}`,
			PaymentMethodPerBytes{
				Price: price,
				Bytes: datasize.Gigabyte,
			},
			nil,
		},
		{
			`{
				"price": {},
				"bytes": 0
			}`,
			PaymentMethodPerBytes{},
			nil,
		},
		{
			`{}`,
			PaymentMethodPerBytes{},
			nil,
		},
	}

	for _, test := range tests {
		var model PaymentMethodPerBytes
		err := json.Unmarshal([]byte(test.json), &model)

		assert.Equal(t, test.expectedModel, model)
		if test.expectedError != nil {
			assert.EqualError(t, err, test.expectedError.Error())
		} else {
			assert.NoError(t, err)
		}
	}
}
