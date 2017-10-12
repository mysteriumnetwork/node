package money

type Money struct {
	Amount   uint64   `json:"amount"`
	Currency Currency `json:"currency"`
}

func NewMoney(amount float64, currency Currency) Money {
	return Money{uint64(amount * 100000000), currency}
}
