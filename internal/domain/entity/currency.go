package entity

type Currency int16

const (
	CURRENCY_UNSPECIFIED Currency = iota
	CURRENCY_USD
	CURRENCY_EUR
	CURRENCY_RUB
	CURRENCY_BYN
)
