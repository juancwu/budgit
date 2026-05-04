// Package currency provides ISO 4217 currency code validation and a small
// curated list of supported codes for UI selection.
package currency

import "strings"

const Default = "CAD"

// supported is the curated list of ISO 4217 codes shown in the account creation
// UI. Add codes here as users request them. Validation accepts any 3-letter
// uppercase code in this list.
var supported = []string{
	"CAD",
	"USD",
	"EUR",
	"GBP",
	"JPY",
	"AUD",
	"CHF",
	"CNY",
	"HKD",
	"MXN",
	"NZD",
	"SGD",
	"INR",
	"BRL",
	"KRW",
	"TWD",
}

func Supported() []string {
	out := make([]string, len(supported))
	copy(out, supported)
	return out
}

// Normalize uppercases and trims the input. It does not validate.
func Normalize(code string) string {
	return strings.ToUpper(strings.TrimSpace(code))
}

// IsValid reports whether code is a supported ISO 4217 code (case-insensitive).
func IsValid(code string) bool {
	c := Normalize(code)
	for _, s := range supported {
		if s == c {
			return true
		}
	}
	return false
}
