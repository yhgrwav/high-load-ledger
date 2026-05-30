package service

import (
	gen "high-load-ledger/gen/go"

	"github.com/google/uuid"
)

type ExistingAccount struct {
	ID       uuid.UUID
	Currency gen.Currency
	Balance  int64
}

type AccountPool map[gen.Currency][]ExistingAccount

func GetValidCurrencies() []gen.Currency {
	result := make([]gen.Currency, 0, len(gen.Currency_value))
	for _, value := range gen.Currency_value {
		if value == int32(gen.Currency_CURRENCY_UNSPECIFIED) {
			continue
		}
		result = append(result, gen.Currency(value))
	}
	return result
}

func (p AccountPool) Total() int {
	total := 0
	for _, accounts := range p {
		total += len(accounts)
	}
	return total
}

func (p AccountPool) CurrenciesWithMinAccounts(min int) []gen.Currency {
	var out []gen.Currency
	for currency, accounts := range p {
		if len(accounts) >= min {
			out = append(out, currency)
		}
	}
	return out
}

func (p AccountPool) CurrenciesWithAccounts() []gen.Currency {
	var out []gen.Currency
	for currency, accounts := range p {
		if len(accounts) > 0 {
			out = append(out, currency)
		}
	}
	return out
}
