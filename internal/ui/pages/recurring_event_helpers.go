package pages

import (
	"fmt"
	"time"

	"git.juancwu.dev/juancwu/budgit/internal/model"
)

func accountLabel(ev *model.RecurringEvent, accountByID map[string]string) string {
	src := accountByID[ev.SourceAccountID]
	if src == "" {
		src = ev.SourceAccountID
	}
	return src
}

var weekdayLabels = []string{"Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"}

func recurrenceSummary(ev *model.RecurringEvent) string {
	timePart := fmt.Sprintf(" at %02d:%02d", ev.FireHour, ev.FireMinute)
	switch ev.Frequency {
	case model.RecurringFrequencyDaily:
		if ev.IntervalCount == 1 {
			return "Daily" + timePart
		}
		return fmt.Sprintf("Every %d days%s", ev.IntervalCount, timePart)
	case model.RecurringFrequencyWeekly:
		dow := ""
		if ev.DayOfWeek != nil && *ev.DayOfWeek >= 0 && *ev.DayOfWeek < len(weekdayLabels) {
			dow = " on " + weekdayLabels[*ev.DayOfWeek]
		}
		if ev.IntervalCount == 1 {
			return "Weekly" + dow + timePart
		}
		return fmt.Sprintf("Every %d weeks%s%s", ev.IntervalCount, dow, timePart)
	case model.RecurringFrequencyMonthly:
		dom := ""
		if ev.DayOfMonth != nil {
			dom = fmt.Sprintf(" on day %d", *ev.DayOfMonth)
		}
		if ev.IntervalCount == 1 {
			return "Monthly" + dom + timePart
		}
		return fmt.Sprintf("Every %d months%s%s", ev.IntervalCount, dom, timePart)
	case model.RecurringFrequencyYearly:
		date := ""
		if ev.MonthOfYear != nil && ev.DayOfMonth != nil {
			date = fmt.Sprintf(" on %s %d", time.Month(*ev.MonthOfYear).String(), *ev.DayOfMonth)
		}
		if ev.IntervalCount == 1 {
			return "Yearly" + date + timePart
		}
		return fmt.Sprintf("Every %d years%s%s", ev.IntervalCount, date, timePart)
	}
	return string(ev.Frequency)
}
