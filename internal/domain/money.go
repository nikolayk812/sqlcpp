package domain

import (
	"github.com/shopspring/decimal"
	"golang.org/x/text/currency"
)

type Money struct {
	Amount   decimal.Decimal
	Currency currency.Unit
}
