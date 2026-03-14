package model

import (
	"fmt"

	"github.com/shopspring/decimal"
)

// FormatMoney formats a decimal as a dollar string like "$12.50"
func FormatMoney(d decimal.Decimal) string {
	return fmt.Sprintf("$%s", d.StringFixed(2))
}

// FormatDecimal formats a decimal for form input values like "12.50"
func FormatDecimal(d decimal.Decimal) string {
	return d.StringFixed(2)
}
