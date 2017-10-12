package money

import (
	"testing"
	"github.com/stretchr/testify/assert"
)

func Test_NewMoney(t *testing.T) {
	assert.Equal(
		t,
		uint64(0.150*100000000),
		NewMoney(0.150, CURRENCY_MYST).Amount,
	)

	assert.Equal(
		t,
		uint64(0*100000000),
		NewMoney(0, CURRENCY_MYST).Amount,
	)

	assert.Equal(
		t,
		uint64(10*100000000),
		NewMoney(10, CURRENCY_MYST).Amount,
	)

	assert.NotEqual(
		t,
		uint64(1),
		NewMoney(1, CURRENCY_MYST).Amount,
	)
}
