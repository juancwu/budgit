package blocks

import "github.com/shopspring/decimal"

func decimalHundred() decimal.Decimal { return decimal.NewFromInt(100) }
func decimalZero() decimal.Decimal    { return decimal.Zero }

func targetDisplay(t *decimal.Decimal) string {
	if t == nil {
		return ""
	}
	return t.StringFixedBank(2)
}
