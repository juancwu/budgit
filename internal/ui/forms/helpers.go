package forms

import (
	"strconv"

	"git.juancwu.dev/juancwu/budgit/internal/misc/timezone"
	"git.juancwu.dev/juancwu/budgit/internal/model"
)

func intToStr(n int) string { return strconv.Itoa(n) }

func kindLabel(v string) string {
	switch v {
	case string(model.RecurringEventKindBill):
		return "Bill (withdrawal)"
	case string(model.RecurringEventKindFund):
		return "Fund (deposit)"
	}
	return ""
}

func frequencyLabel(v string) string {
	switch v {
	case string(model.RecurringFrequencyDaily):
		return "Daily"
	case string(model.RecurringFrequencyWeekly):
		return "Weekly"
	case string(model.RecurringFrequencyMonthly):
		return "Monthly"
	case string(model.RecurringFrequencyYearly):
		return "Yearly"
	}
	return ""
}

func weekdayLabel(v string) string {
	i, err := strconv.Atoi(v)
	if err != nil || i < 0 || i >= len(weekdayNames) {
		return ""
	}
	return weekdayNames[i]
}

func monthLabel(v string) string {
	i, err := strconv.Atoi(v)
	if err != nil || i < 1 || i > len(monthNames) {
		return ""
	}
	return monthNames[i-1]
}

func accountLabel(accounts []*model.Account, id string) string {
	for _, a := range accounts {
		if a.ID == id {
			return a.Name
		}
	}
	return ""
}

func timezoneLabel(tzs []timezone.TimezoneOption, v string) string {
	for _, tz := range tzs {
		if tz.Value == v {
			return tz.Label
		}
	}
	return ""
}
