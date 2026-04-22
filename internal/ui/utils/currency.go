package utils

import "strings"

func FormatDecimalWithThousands(numStr string) (string, error) {
	// Split into integer and decimal parts
	parts := strings.SplitN(numStr, ".", 2)
	intPart := parts[0]

	// Handle negative numbers
	negative := false
	if strings.HasPrefix(intPart, "-") {
		negative = true
		intPart = intPart[1:]
	}

	// Insert thousand separators
	var result []byte
	for i, c := range intPart {
		if i > 0 && (len(intPart)-i)%3 == 0 {
			result = append(result, ',')
		}
		result = append(result, byte(c))
	}

	// Reassemble
	formatted := string(result)
	if len(parts) == 2 {
		formatted += "." + parts[1]
	}
	if negative {
		formatted = "-" + formatted
	}
	return formatted, nil
}
