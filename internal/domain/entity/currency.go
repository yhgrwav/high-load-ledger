package entity

type Currency int16

const (
	CURRENCY_UNSPECIFIED Currency = iota
	CURRENCY_USD
	CURRENCY_EUR
	CURRENCY_RUB
	CURRENCY_BYN
)

func (c Currency) IsValid() bool {
	switch c {
	case CURRENCY_USD, CURRENCY_EUR, CURRENCY_RUB, CURRENCY_BYN:
		return true
	default:
		return false
	}
}
