package timezone

type TimezoneOption struct {
	Value string
	Label string
}

func CommonTimezones() []TimezoneOption {
	return []TimezoneOption{
		{Value: "UTC", Label: "UTC"},
		{Value: "America/New_York", Label: "Eastern Time (US & Canada)"},
		{Value: "America/Chicago", Label: "Central Time (US & Canada)"},
		{Value: "America/Denver", Label: "Mountain Time (US & Canada)"},
		{Value: "America/Los_Angeles", Label: "Pacific Time (US & Canada)"},
		{Value: "America/Anchorage", Label: "Alaska"},
		{Value: "Pacific/Honolulu", Label: "Hawaii"},
		{Value: "America/Phoenix", Label: "Arizona"},
		{Value: "America/Toronto", Label: "Toronto"},
		{Value: "America/Vancouver", Label: "Vancouver"},
		{Value: "America/Edmonton", Label: "Edmonton"},
		{Value: "America/Winnipeg", Label: "Winnipeg"},
		{Value: "America/Halifax", Label: "Atlantic Time (Canada)"},
		{Value: "America/St_Johns", Label: "Newfoundland"},
		{Value: "America/Mexico_City", Label: "Mexico City"},
		{Value: "America/Bogota", Label: "Bogota"},
		{Value: "America/Lima", Label: "Lima"},
		{Value: "America/Santiago", Label: "Santiago"},
		{Value: "America/Argentina/Buenos_Aires", Label: "Buenos Aires"},
		{Value: "America/Sao_Paulo", Label: "Sao Paulo"},
		{Value: "Europe/London", Label: "London"},
		{Value: "Europe/Paris", Label: "Paris"},
		{Value: "Europe/Berlin", Label: "Berlin"},
		{Value: "Europe/Madrid", Label: "Madrid"},
		{Value: "Europe/Rome", Label: "Rome"},
		{Value: "Europe/Amsterdam", Label: "Amsterdam"},
		{Value: "Europe/Zurich", Label: "Zurich"},
		{Value: "Europe/Stockholm", Label: "Stockholm"},
		{Value: "Europe/Helsinki", Label: "Helsinki"},
		{Value: "Europe/Warsaw", Label: "Warsaw"},
		{Value: "Europe/Istanbul", Label: "Istanbul"},
		{Value: "Europe/Moscow", Label: "Moscow"},
		{Value: "Asia/Dubai", Label: "Dubai"},
		{Value: "Asia/Kolkata", Label: "India (Kolkata)"},
		{Value: "Asia/Bangkok", Label: "Bangkok"},
		{Value: "Asia/Singapore", Label: "Singapore"},
		{Value: "Asia/Hong_Kong", Label: "Hong Kong"},
		{Value: "Asia/Shanghai", Label: "Shanghai"},
		{Value: "Asia/Tokyo", Label: "Tokyo"},
		{Value: "Asia/Seoul", Label: "Seoul"},
		{Value: "Australia/Sydney", Label: "Sydney"},
		{Value: "Australia/Melbourne", Label: "Melbourne"},
		{Value: "Australia/Perth", Label: "Perth"},
		{Value: "Australia/Brisbane", Label: "Brisbane"},
		{Value: "Pacific/Auckland", Label: "Auckland"},
		{Value: "Africa/Cairo", Label: "Cairo"},
		{Value: "Africa/Johannesburg", Label: "Johannesburg"},
		{Value: "Africa/Lagos", Label: "Lagos"},
	}
}
