package money

type Money struct {
	Amount   uint64   `json:"amount,omitempty"`
	Currency Currency `json:"currency,omitempty"`
}

func NewMoney(amount float64, currency Currency) Money {
	return Money{uint64(amount * 100000000), currency}
}
