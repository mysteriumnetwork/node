package session

import (
	"testing"
	"time"

	"github.com/mysteriumnetwork/node/money"
	"github.com/mysteriumnetwork/node/services/openvpn/discovery/dto"
	"github.com/stretchr/testify/assert"
)

func Test_CorrectMoneyValueIsReturnedForTotalAmount(t *testing.T) {
	aCalc := AmountCalc{
		PaymentDef: dto.PaymentPerTime{
			Duration: time.Minute,
			Price: money.Money{
				Amount:   100,
				Currency: money.CURRENCY_MYST,
			},
		},
	}

	elapsed := time.Minute*3 + time.Second*15

	totalAmount := aCalc.TotalAmount(elapsed)

	assert.Equal(t, uint64(300), totalAmount.Amount)
}
