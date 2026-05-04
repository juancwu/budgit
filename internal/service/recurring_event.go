package service

import (
	"fmt"
	"log/slog"
	"strings"
	"time"

	"git.juancwu.dev/juancwu/budgit/internal/model"
	"git.juancwu.dev/juancwu/budgit/internal/repository"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// RecurringEventService manages recurring bills, funds, and transfers and
// materializes due events into actual transactions via TransactionService.
type RecurringEventService struct {
	repo           repository.RecurringEventRepository
	txService      *TransactionService
	accountService *AccountService
}

func NewRecurringEventService(
	repo repository.RecurringEventRepository,
	txService *TransactionService,
	accountService *AccountService,
) *RecurringEventService {
	return &RecurringEventService{
		repo:           repo,
		txService:      txService,
		accountService: accountService,
	}
}

type CreateRecurringEventInput struct {
	SpaceID         string
	Kind            model.RecurringEventKind
	SourceAccountID string
	Title           string
	Amount          decimal.Decimal
	Description     string

	Frequency     model.RecurringFrequency
	IntervalCount int
	DayOfWeek     *int
	DayOfMonth    *int
	MonthOfYear   *int
	FireHour      int
	FireMinute    int
	Timezone      string

	// StartDate is the local calendar date (Y-M-D) of the first intended firing
	// in the event's timezone. The first NextRunAt is computed from StartDate,
	// FireHour, FireMinute, and Timezone — clamped to the recurrence anchors.
	StartDate time.Time
}

func (s *RecurringEventService) Create(input CreateRecurringEventInput) (*model.RecurringEvent, error) {
	if err := validateRule(input.Kind, input.SourceAccountID, input.Frequency, input.IntervalCount, input.DayOfWeek, input.DayOfMonth, input.MonthOfYear, input.FireHour, input.FireMinute, input.Timezone); err != nil {
		return nil, err
	}
	title := strings.TrimSpace(input.Title)
	if title == "" {
		return nil, fmt.Errorf("title is required")
	}
	if !input.Amount.IsPositive() {
		return nil, fmt.Errorf("amount must be greater than zero")
	}
	if input.SpaceID == "" {
		return nil, fmt.Errorf("space id is required")
	}

	loc, err := time.LoadLocation(input.Timezone)
	if err != nil {
		return nil, fmt.Errorf("invalid timezone: %w", err)
	}
	if input.StartDate.IsZero() {
		return nil, fmt.Errorf("start date is required")
	}

	firstFire, err := firstFireOnOrAfter(input.Frequency, input.IntervalCount, input.DayOfWeek, input.DayOfMonth, input.MonthOfYear, input.FireHour, input.FireMinute, loc, input.StartDate)
	if err != nil {
		return nil, err
	}

	var description *string
	if d := strings.TrimSpace(input.Description); d != "" {
		description = &d
	}

	now := time.Now().UTC()
	ev := &model.RecurringEvent{
		ID:              uuid.NewString(),
		SpaceID:         input.SpaceID,
		Kind:            input.Kind,
		SourceAccountID: input.SourceAccountID,
		Title:           title,
		Amount:          input.Amount,
		Description:     description,
		Frequency:       input.Frequency,
		IntervalCount:   input.IntervalCount,
		DayOfWeek:       input.DayOfWeek,
		DayOfMonth:      input.DayOfMonth,
		MonthOfYear:     input.MonthOfYear,
		FireHour:        input.FireHour,
		FireMinute:      input.FireMinute,
		Timezone:        input.Timezone,
		NextRunAt:       firstFire.UTC(),
		Paused:          false,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if err := s.repo.Create(ev); err != nil {
		return nil, fmt.Errorf("failed to create recurring event: %w", err)
	}
	return ev, nil
}

type UpdateRecurringEventInput struct {
	ID              string
	Kind            model.RecurringEventKind
	SourceAccountID string
	Title           string
	Amount          decimal.Decimal
	Description     string

	Frequency     model.RecurringFrequency
	IntervalCount int
	DayOfWeek     *int
	DayOfMonth    *int
	MonthOfYear   *int
	FireHour      int
	FireMinute    int
	Timezone      string

	// StartDate, if non-zero, recomputes the next firing. If zero, the current
	// cursor is kept (useful for purely cosmetic edits like renaming).
	StartDate time.Time
}

func (s *RecurringEventService) Update(input UpdateRecurringEventInput) (*model.RecurringEvent, error) {
	if err := validateRule(input.Kind, input.SourceAccountID, input.Frequency, input.IntervalCount, input.DayOfWeek, input.DayOfMonth, input.MonthOfYear, input.FireHour, input.FireMinute, input.Timezone); err != nil {
		return nil, err
	}
	title := strings.TrimSpace(input.Title)
	if title == "" {
		return nil, fmt.Errorf("title is required")
	}
	if !input.Amount.IsPositive() {
		return nil, fmt.Errorf("amount must be greater than zero")
	}

	existing, err := s.repo.ByID(input.ID)
	if err != nil {
		return nil, err
	}

	loc, err := time.LoadLocation(input.Timezone)
	if err != nil {
		return nil, fmt.Errorf("invalid timezone: %w", err)
	}

	nextRun := existing.NextRunAt
	if !input.StartDate.IsZero() {
		firstFire, err := firstFireOnOrAfter(input.Frequency, input.IntervalCount, input.DayOfWeek, input.DayOfMonth, input.MonthOfYear, input.FireHour, input.FireMinute, loc, input.StartDate)
		if err != nil {
			return nil, err
		}
		nextRun = firstFire.UTC()
	}

	var description *string
	if d := strings.TrimSpace(input.Description); d != "" {
		description = &d
	}

	existing.Kind = input.Kind
	existing.SourceAccountID = input.SourceAccountID
	existing.Title = title
	existing.Amount = input.Amount
	existing.Description = description
	existing.Frequency = input.Frequency
	existing.IntervalCount = input.IntervalCount
	existing.DayOfWeek = input.DayOfWeek
	existing.DayOfMonth = input.DayOfMonth
	existing.MonthOfYear = input.MonthOfYear
	existing.FireHour = input.FireHour
	existing.FireMinute = input.FireMinute
	existing.Timezone = input.Timezone
	existing.NextRunAt = nextRun

	if err := s.repo.Update(existing); err != nil {
		return nil, err
	}
	return existing, nil
}

func (s *RecurringEventService) Delete(id string) error {
	return s.repo.Delete(id)
}

func (s *RecurringEventService) SetPaused(id string, paused bool) error {
	return s.repo.SetPaused(id, paused)
}

func (s *RecurringEventService) Get(id string) (*model.RecurringEvent, error) {
	return s.repo.ByID(id)
}

func (s *RecurringEventService) ListBySpace(spaceID string) ([]*model.RecurringEvent, error) {
	return s.repo.BySpaceID(spaceID)
}

func (s *RecurringEventService) ListByAccount(accountID string) ([]*model.RecurringEvent, error) {
	return s.repo.ByAccountID(accountID)
}

// ProcessDue materializes every recurring event whose next_run_at is at or
// before `now`, advancing each cursor and backfilling missed occurrences. One
// event's failure is logged but does not stop processing of others.
func (s *RecurringEventService) ProcessDue(now time.Time) error {
	events, err := s.repo.DueBefore(now)
	if err != nil {
		return fmt.Errorf("failed to list due events: %w", err)
	}
	for _, ev := range events {
		if err := s.fireUntilCaughtUp(ev, now); err != nil {
			slog.Error("recurring event materialization failed",
				"error", err, "event_id", ev.ID, "kind", ev.Kind)
		}
	}
	return nil
}

func (s *RecurringEventService) fireUntilCaughtUp(ev *model.RecurringEvent, now time.Time) error {
	loc, err := time.LoadLocation(ev.Timezone)
	if err != nil {
		return fmt.Errorf("invalid timezone %q: %w", ev.Timezone, err)
	}
	for !ev.NextRunAt.After(now) {
		if err := s.materialize(ev); err != nil {
			return fmt.Errorf("materialize: %w", err)
		}
		next, err := nextFireAfter(ev, ev.NextRunAt, loc)
		if err != nil {
			return fmt.Errorf("compute next: %w", err)
		}
		last := ev.NextRunAt
		if err := s.repo.UpdateCursor(ev.ID, next, last); err != nil {
			return fmt.Errorf("persist cursor: %w", err)
		}
		ev.LastRunAt = &last
		ev.NextRunAt = next
	}
	return nil
}

func (s *RecurringEventService) materialize(ev *model.RecurringEvent) error {
	desc := ""
	if ev.Description != nil {
		desc = *ev.Description
	}
	switch ev.Kind {
	case model.RecurringEventKindBill:
		_, err := s.txService.PayBill(PayBillInput{
			AccountID:   ev.SourceAccountID,
			Title:       ev.Title,
			Amount:      ev.Amount,
			OccurredAt:  ev.NextRunAt,
			Description: desc,
		})
		return err
	case model.RecurringEventKindFund:
		_, err := s.txService.Deposit(DepositInput{
			AccountID:   ev.SourceAccountID,
			Title:       ev.Title,
			Amount:      ev.Amount,
			OccurredAt:  ev.NextRunAt,
			Description: desc,
		})
		return err
	}
	return fmt.Errorf("unknown recurring event kind: %s", ev.Kind)
}

// ----- Recurrence math -----

func validateRule(kind model.RecurringEventKind, src string, freq model.RecurringFrequency, interval int, dow, dom, moy *int, hour, minute int, tz string) error {
	switch kind {
	case model.RecurringEventKindBill, model.RecurringEventKindFund:
		// ok
	default:
		return fmt.Errorf("invalid kind: %s", kind)
	}
	if src == "" {
		return fmt.Errorf("source account is required")
	}
	if interval < 1 {
		return fmt.Errorf("interval must be at least 1")
	}
	if hour < 0 || hour > 23 || minute < 0 || minute > 59 {
		return fmt.Errorf("invalid time of day")
	}
	if tz == "" {
		return fmt.Errorf("timezone is required")
	}
	if _, err := time.LoadLocation(tz); err != nil {
		return fmt.Errorf("invalid timezone: %w", err)
	}
	switch freq {
	case model.RecurringFrequencyDaily:
		// no anchors required
	case model.RecurringFrequencyWeekly:
		if dow == nil || *dow < 0 || *dow > 6 {
			return fmt.Errorf("weekly events require day of week")
		}
	case model.RecurringFrequencyMonthly:
		if dom == nil || *dom < 1 || *dom > 31 {
			return fmt.Errorf("monthly events require day of month")
		}
	case model.RecurringFrequencyYearly:
		if dom == nil || *dom < 1 || *dom > 31 {
			return fmt.Errorf("yearly events require day of month")
		}
		if moy == nil || *moy < 1 || *moy > 12 {
			return fmt.Errorf("yearly events require month of year")
		}
	default:
		return fmt.Errorf("invalid frequency: %s", freq)
	}
	return nil
}

// firstFireOnOrAfter computes the first firing in `loc` at or after the local
// midnight of startDate, snapped to the recurrence anchors and time-of-day.
func firstFireOnOrAfter(freq model.RecurringFrequency, interval int, dow, dom, moy *int, hour, minute int, loc *time.Location, startDate time.Time) (time.Time, error) {
	startLocal := time.Date(startDate.Year(), startDate.Month(), startDate.Day(), 0, 0, 0, 0, loc)
	threshold := startLocal.Add(-time.Nanosecond) // nextFireAfter computes strictly-after; -1ns lets the first candidate land on startDate
	ev := &model.RecurringEvent{
		Frequency:     freq,
		IntervalCount: interval,
		DayOfWeek:     dow,
		DayOfMonth:    dom,
		MonthOfYear:   moy,
		FireHour:      hour,
		FireMinute:    minute,
	}
	return nextFireAfter(ev, threshold, loc)
}

// nextFireAfter computes the next firing strictly after `after`, in UTC.
func nextFireAfter(ev *model.RecurringEvent, after time.Time, loc *time.Location) (time.Time, error) {
	afterLocal := after.In(loc)
	switch ev.Frequency {
	case model.RecurringFrequencyDaily:
		c := time.Date(afterLocal.Year(), afterLocal.Month(), afterLocal.Day(), ev.FireHour, ev.FireMinute, 0, 0, loc)
		for !c.After(after) {
			c = c.AddDate(0, 0, ev.IntervalCount)
		}
		return c.UTC(), nil

	case model.RecurringFrequencyWeekly:
		if ev.DayOfWeek == nil {
			return time.Time{}, fmt.Errorf("weekly event missing day of week")
		}
		c := time.Date(afterLocal.Year(), afterLocal.Month(), afterLocal.Day(), ev.FireHour, ev.FireMinute, 0, 0, loc)
		shift := (int(time.Weekday(*ev.DayOfWeek)) - int(c.Weekday()) + 7) % 7
		c = c.AddDate(0, 0, shift)
		for !c.After(after) {
			c = c.AddDate(0, 0, 7*ev.IntervalCount)
		}
		return c.UTC(), nil

	case model.RecurringFrequencyMonthly:
		if ev.DayOfMonth == nil {
			return time.Time{}, fmt.Errorf("monthly event missing day of month")
		}
		y, m := afterLocal.Year(), afterLocal.Month()
		c := monthlyCandidate(y, m, *ev.DayOfMonth, ev.FireHour, ev.FireMinute, loc)
		for !c.After(after) {
			y, m = addMonths(y, m, ev.IntervalCount)
			c = monthlyCandidate(y, m, *ev.DayOfMonth, ev.FireHour, ev.FireMinute, loc)
		}
		return c.UTC(), nil

	case model.RecurringFrequencyYearly:
		if ev.DayOfMonth == nil || ev.MonthOfYear == nil {
			return time.Time{}, fmt.Errorf("yearly event missing anchors")
		}
		moy := time.Month(*ev.MonthOfYear)
		y := afterLocal.Year()
		c := monthlyCandidate(y, moy, *ev.DayOfMonth, ev.FireHour, ev.FireMinute, loc)
		for !c.After(after) {
			y += ev.IntervalCount
			c = monthlyCandidate(y, moy, *ev.DayOfMonth, ev.FireHour, ev.FireMinute, loc)
		}
		return c.UTC(), nil
	}
	return time.Time{}, fmt.Errorf("unknown frequency: %s", ev.Frequency)
}

// monthlyCandidate constructs a fire time at year/month, clamping the day to
// that month's actual length (e.g. day=31 in February becomes the 28th/29th).
func monthlyCandidate(year int, month time.Month, dom, hour, minute int, loc *time.Location) time.Time {
	last := lastDayOfMonth(year, month, loc)
	if dom > last {
		dom = last
	}
	return time.Date(year, month, dom, hour, minute, 0, 0, loc)
}

func lastDayOfMonth(year int, month time.Month, loc *time.Location) int {
	firstOfNext := time.Date(year, month+1, 1, 0, 0, 0, 0, loc)
	return firstOfNext.AddDate(0, 0, -1).Day()
}

func addMonths(y int, m time.Month, n int) (int, time.Month) {
	total := (int(m) - 1) + n
	years := total / 12
	rem := total % 12
	if rem < 0 {
		rem += 12
		years--
	}
	return y + years, time.Month(rem + 1)
}
