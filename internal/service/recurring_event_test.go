package service

import (
	"testing"
	"time"

	"git.juancwu.dev/juancwu/budgit/internal/model"
)

func mustLoad(t *testing.T, name string) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation(name)
	if err != nil {
		t.Fatalf("LoadLocation(%q): %v", name, err)
	}
	return loc
}

func intPtr(v int) *int { return &v }

func TestNextFireAfter_Daily(t *testing.T) {
	loc := mustLoad(t, "America/New_York")
	ev := &model.RecurringEvent{
		Frequency:     model.RecurringFrequencyDaily,
		IntervalCount: 1,
		FireHour:      9,
		FireMinute:    30,
	}
	// "after" = 2026-05-01 14:00 NY → next fire same day's 09:30 already passed,
	// so should be 2026-05-02 09:30 NY.
	after := time.Date(2026, 5, 1, 14, 0, 0, 0, loc)
	got, err := nextFireAfter(ev, after, loc)
	if err != nil {
		t.Fatal(err)
	}
	want := time.Date(2026, 5, 2, 9, 30, 0, 0, loc)
	if !got.Equal(want) {
		t.Errorf("daily next: got %v want %v", got.In(loc), want)
	}
}

func TestNextFireAfter_DailyEvery3Days(t *testing.T) {
	loc := mustLoad(t, "UTC")
	ev := &model.RecurringEvent{
		Frequency:     model.RecurringFrequencyDaily,
		IntervalCount: 3,
		FireHour:      8,
		FireMinute:    0,
	}
	after := time.Date(2026, 5, 1, 8, 0, 1, 0, loc) // 1 second past today's fire
	got, _ := nextFireAfter(ev, after, loc)
	want := time.Date(2026, 5, 4, 8, 0, 0, 0, loc)
	if !got.Equal(want) {
		t.Errorf("got %v want %v", got, want)
	}
}

func TestNextFireAfter_Weekly(t *testing.T) {
	loc := mustLoad(t, "UTC")
	// Every Tuesday (DayOfWeek=2) at 10:00, after Wed 2026-05-06.
	dow := 2
	ev := &model.RecurringEvent{
		Frequency:     model.RecurringFrequencyWeekly,
		IntervalCount: 1,
		DayOfWeek:     &dow,
		FireHour:      10,
		FireMinute:    0,
	}
	after := time.Date(2026, 5, 6, 12, 0, 0, 0, loc) // Wed 12:00
	got, _ := nextFireAfter(ev, after, loc)
	want := time.Date(2026, 5, 12, 10, 0, 0, 0, loc) // following Tue
	if !got.Equal(want) {
		t.Errorf("got %v want %v", got, want)
	}
}

func TestNextFireAfter_MonthlyDayClamp(t *testing.T) {
	loc := mustLoad(t, "UTC")
	// Every month on day 31. Jan 31 → Feb 28 (2026 not leap).
	dom := 31
	ev := &model.RecurringEvent{
		Frequency:     model.RecurringFrequencyMonthly,
		IntervalCount: 1,
		DayOfMonth:    &dom,
		FireHour:      0,
		FireMinute:    0,
	}
	after := time.Date(2026, 1, 31, 0, 0, 1, 0, loc)
	got, _ := nextFireAfter(ev, after, loc)
	want := time.Date(2026, 2, 28, 0, 0, 0, 0, loc)
	if !got.Equal(want) {
		t.Errorf("got %v want %v", got, want)
	}
	// Then Feb 28 → Mar 31 (anchor preserved).
	got2, _ := nextFireAfter(ev, want, loc)
	want2 := time.Date(2026, 3, 31, 0, 0, 0, 0, loc)
	if !got2.Equal(want2) {
		t.Errorf("got2 %v want2 %v", got2, want2)
	}
}

func TestNextFireAfter_Yearly(t *testing.T) {
	loc := mustLoad(t, "UTC")
	dom := 15
	moy := 6
	ev := &model.RecurringEvent{
		Frequency:     model.RecurringFrequencyYearly,
		IntervalCount: 1,
		DayOfMonth:    &dom,
		MonthOfYear:   &moy,
		FireHour:      12,
		FireMinute:    0,
	}
	after := time.Date(2026, 6, 15, 12, 0, 1, 0, loc)
	got, _ := nextFireAfter(ev, after, loc)
	want := time.Date(2027, 6, 15, 12, 0, 0, 0, loc)
	if !got.Equal(want) {
		t.Errorf("got %v want %v", got, want)
	}
}

func TestNextFireAfter_TimezoneCrossesUTCBoundary(t *testing.T) {
	loc := mustLoad(t, "America/Los_Angeles")
	// 09:00 LA daily. From 2026-05-01 17:00 UTC (10:00 LA), already past, so
	// next is 2026-05-02 09:00 LA = 16:00 UTC.
	ev := &model.RecurringEvent{
		Frequency:     model.RecurringFrequencyDaily,
		IntervalCount: 1,
		FireHour:      9,
		FireMinute:    0,
	}
	after := time.Date(2026, 5, 1, 17, 0, 0, 0, time.UTC)
	got, _ := nextFireAfter(ev, after, loc)
	want := time.Date(2026, 5, 2, 16, 0, 0, 0, time.UTC) // PDT, UTC-7
	if !got.Equal(want) {
		t.Errorf("got %v want %v", got, want)
	}
}

func TestFirstFireOnOrAfter_SameDayBeforeFire(t *testing.T) {
	loc := mustLoad(t, "UTC")
	// Daily at 09:00, start date 2026-05-10 → first fire 2026-05-10 09:00.
	got, err := firstFireOnOrAfter(model.RecurringFrequencyDaily, 1, nil, nil, nil, 9, 0, loc, time.Date(2026, 5, 10, 0, 0, 0, 0, loc))
	if err != nil {
		t.Fatal(err)
	}
	want := time.Date(2026, 5, 10, 9, 0, 0, 0, loc)
	if !got.Equal(want) {
		t.Errorf("got %v want %v", got, want)
	}
}

func TestFirstFireOnOrAfter_WeeklyShiftsToTargetDayOfWeek(t *testing.T) {
	loc := mustLoad(t, "UTC")
	// Start 2026-05-04 (Mon), target weekday Friday (5) → first fire 2026-05-08.
	got, _ := firstFireOnOrAfter(model.RecurringFrequencyWeekly, 1, intPtr(5), nil, nil, 8, 0, loc, time.Date(2026, 5, 4, 0, 0, 0, 0, loc))
	want := time.Date(2026, 5, 8, 8, 0, 0, 0, loc)
	if !got.Equal(want) {
		t.Errorf("got %v want %v", got, want)
	}
}

func TestAddMonths(t *testing.T) {
	tests := []struct {
		y int
		m time.Month
		n int
		wy int
		wm time.Month
	}{
		{2026, time.January, 1, 2026, time.February},
		{2026, time.December, 1, 2027, time.January},
		{2026, time.November, 3, 2027, time.February},
		{2026, time.January, 12, 2027, time.January},
		{2026, time.January, 25, 2028, time.February},
	}
	for _, tt := range tests {
		gy, gm := addMonths(tt.y, tt.m, tt.n)
		if gy != tt.wy || gm != tt.wm {
			t.Errorf("addMonths(%d,%v,%d) = %d,%v; want %d,%v", tt.y, tt.m, tt.n, gy, gm, tt.wy, tt.wm)
		}
	}
}
