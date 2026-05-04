package handler

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"git.juancwu.dev/juancwu/budgit/internal/misc/timezone"
	"git.juancwu.dev/juancwu/budgit/internal/model"
	"git.juancwu.dev/juancwu/budgit/internal/repository"
	"git.juancwu.dev/juancwu/budgit/internal/routeurl"
	"git.juancwu.dev/juancwu/budgit/internal/service"
	"git.juancwu.dev/juancwu/budgit/internal/ui"
	"git.juancwu.dev/juancwu/budgit/internal/ui/forms"
	"git.juancwu.dev/juancwu/budgit/internal/ui/pages"
	"github.com/shopspring/decimal"
)

type recurringEventHandler struct {
	recurringService *service.RecurringEventService
	accountService   *service.AccountService
	spaceService     *service.SpaceService
}

func NewRecurringEventHandler(rec *service.RecurringEventService, acc *service.AccountService, sp *service.SpaceService) *recurringEventHandler {
	return &recurringEventHandler{recurringService: rec, accountService: acc, spaceService: sp}
}

// ListPage shows every recurring event for a space.
func (h *recurringEventHandler) ListPage(w http.ResponseWriter, r *http.Request) {
	spaceID := r.PathValue("spaceID")
	space, err := h.spaceService.GetSpace(spaceID)
	if err != nil {
		ui.Render(w, r, pages.NotFound())
		return
	}
	events, err := h.recurringService.ListBySpace(spaceID)
	if err != nil {
		slog.Error("failed to list recurring events", "error", err, "space_id", spaceID)
		ui.RenderError(w, r, "Failed to load recurring events", http.StatusInternalServerError)
		return
	}
	accounts, err := h.accountService.GetAccountsForSpace(spaceID)
	if err != nil {
		slog.Error("failed to load accounts", "error", err, "space_id", spaceID)
		ui.RenderError(w, r, "Failed to load recurring events", http.StatusInternalServerError)
		return
	}
	accountByID := map[string]string{}
	for _, a := range accounts {
		accountByID[a.ID] = a.Name
	}
	ui.Render(w, r, pages.SpaceRecurringEventsPage(pages.SpaceRecurringEventsPageProps{
		SpaceID:     spaceID,
		SpaceName:   space.Name,
		Events:      events,
		AccountByID: accountByID,
	}))
}

// CreatePage shows the create form.
func (h *recurringEventHandler) CreatePage(w http.ResponseWriter, r *http.Request) {
	spaceID := r.PathValue("spaceID")
	space, err := h.spaceService.GetSpace(spaceID)
	if err != nil {
		ui.Render(w, r, pages.NotFound())
		return
	}
	accounts, err := h.accountService.GetAccountsForSpace(spaceID)
	if err != nil {
		slog.Error("failed to load accounts", "error", err, "space_id", spaceID)
		ui.RenderError(w, r, "Failed to load form", http.StatusInternalServerError)
		return
	}

	now := time.Now()
	formProps := forms.RecurringEventFormProps{
		SpaceID:       spaceID,
		Action:        routeurl.URL("action.app.spaces.space.recurring.create", "spaceID", spaceID),
		CancelHref:    routeurl.URL("page.app.spaces.space.recurring", "spaceID", spaceID),
		SubmitLabel:   "Create",
		Accounts:      accounts,
		Timezones:     timezone.CommonTimezones(),
		Kind:          string(model.RecurringEventKindBill),
		Frequency:     string(model.RecurringFrequencyMonthly),
		IntervalCount: "1",
		FireTime:      "09:00",
		Timezone:      "UTC",
		StartDate:     now.Format("2006-01-02"),
		DayOfMonth:    strconv.Itoa(now.Day()),
		DayOfWeek:     strconv.Itoa(int(now.Weekday())),
		MonthOfYear:   strconv.Itoa(int(now.Month())),
	}

	ui.Render(w, r, pages.SpaceCreateRecurringEventPage(pages.SpaceCreateRecurringEventPageProps{
		SpaceID:   spaceID,
		SpaceName: space.Name,
		Form:      formProps,
	}))
}

// EditPage shows the edit form for an existing event.
func (h *recurringEventHandler) EditPage(w http.ResponseWriter, r *http.Request) {
	spaceID := r.PathValue("spaceID")
	eventID := r.PathValue("eventID")

	space, err := h.spaceService.GetSpace(spaceID)
	if err != nil {
		ui.Render(w, r, pages.NotFound())
		return
	}
	ev, err := h.recurringService.Get(eventID)
	if err != nil || ev.SpaceID != spaceID {
		ui.Render(w, r, pages.NotFound())
		return
	}
	accounts, err := h.accountService.GetAccountsForSpace(spaceID)
	if err != nil {
		slog.Error("failed to load accounts", "error", err, "space_id", spaceID)
		ui.RenderError(w, r, "Failed to load form", http.StatusInternalServerError)
		return
	}

	formProps := forms.RecurringEventFormProps{
		SpaceID:         spaceID,
		Action:          routeurl.URL("action.app.spaces.space.recurring.event.edit", "spaceID", spaceID, "eventID", eventID),
		CancelHref:      routeurl.URL("page.app.spaces.space.recurring", "spaceID", spaceID),
		SubmitLabel:     "Save",
		Accounts:        accounts,
		Timezones:       timezone.CommonTimezones(),
		Title:           ev.Title,
		Kind:            string(ev.Kind),
		SourceAccountID: ev.SourceAccountID,
		Amount:          ev.Amount.StringFixedBank(2),
		Frequency:       string(ev.Frequency),
		IntervalCount:   strconv.Itoa(ev.IntervalCount),
		FireTime:        formatTimeOfDay(ev.FireHour, ev.FireMinute),
		Timezone:        ev.Timezone,
		StartDate:       ev.NextRunAt.In(mustLoc(ev.Timezone)).Format("2006-01-02"),
	}
	if ev.Description != nil {
		formProps.Description = *ev.Description
	}
	if ev.DayOfWeek != nil {
		formProps.DayOfWeek = strconv.Itoa(*ev.DayOfWeek)
	}
	if ev.DayOfMonth != nil {
		formProps.DayOfMonth = strconv.Itoa(*ev.DayOfMonth)
	}
	if ev.MonthOfYear != nil {
		formProps.MonthOfYear = strconv.Itoa(*ev.MonthOfYear)
	}

	ui.Render(w, r, pages.SpaceEditRecurringEventPage(pages.SpaceEditRecurringEventPageProps{
		SpaceID:   spaceID,
		SpaceName: space.Name,
		EventID:   eventID,
		Form:      formProps,
	}))
}

// HandleCreate processes the create form submission.
func (h *recurringEventHandler) HandleCreate(w http.ResponseWriter, r *http.Request) {
	spaceID := r.PathValue("spaceID")
	if _, err := h.spaceService.GetSpace(spaceID); err != nil {
		ui.RenderError(w, r, "Space not found", http.StatusNotFound)
		return
	}

	parsed, formProps := h.parseForm(r, spaceID)
	formProps.Action = routeurl.URL("action.app.spaces.space.recurring.create", "spaceID", spaceID)
	formProps.CancelHref = routeurl.URL("page.app.spaces.space.recurring", "spaceID", spaceID)
	formProps.SubmitLabel = "Create"

	if formProps.HasError() {
		ui.Render(w, r, forms.RecurringEventForm(formProps))
		return
	}

	if _, err := h.recurringService.Create(parsed); err != nil {
		slog.Error("failed to create recurring event", "error", err, "space_id", spaceID)
		formProps.GeneralErr = friendlyRecurringError(err)
		ui.Render(w, r, forms.RecurringEventForm(formProps))
		return
	}

	w.Header().Set("HX-Redirect", routeurl.URL("page.app.spaces.space.recurring", "spaceID", spaceID))
	w.WriteHeader(http.StatusOK)
}

// HandleEdit processes the edit form submission.
func (h *recurringEventHandler) HandleEdit(w http.ResponseWriter, r *http.Request) {
	spaceID := r.PathValue("spaceID")
	eventID := r.PathValue("eventID")

	existing, err := h.recurringService.Get(eventID)
	if err != nil || existing.SpaceID != spaceID {
		ui.RenderError(w, r, "Recurring event not found", http.StatusNotFound)
		return
	}

	parsed, formProps := h.parseForm(r, spaceID)
	formProps.Action = routeurl.URL("action.app.spaces.space.recurring.event.edit", "spaceID", spaceID, "eventID", eventID)
	formProps.CancelHref = routeurl.URL("page.app.spaces.space.recurring", "spaceID", spaceID)
	formProps.SubmitLabel = "Save"

	if formProps.HasError() {
		ui.Render(w, r, forms.RecurringEventForm(formProps))
		return
	}

	if _, err := h.recurringService.Update(service.UpdateRecurringEventInput{
		ID:              eventID,
		Kind:            parsed.Kind,
		SourceAccountID: parsed.SourceAccountID,
		Title:           parsed.Title,
		Amount:          parsed.Amount,
		Description:     parsed.Description,
		Frequency:       parsed.Frequency,
		IntervalCount:   parsed.IntervalCount,
		DayOfWeek:       parsed.DayOfWeek,
		DayOfMonth:      parsed.DayOfMonth,
		MonthOfYear:     parsed.MonthOfYear,
		FireHour:        parsed.FireHour,
		FireMinute:      parsed.FireMinute,
		Timezone:        parsed.Timezone,
		StartDate:       parsed.StartDate,
	}); err != nil {
		slog.Error("failed to update recurring event", "error", err, "event_id", eventID)
		formProps.GeneralErr = friendlyRecurringError(err)
		ui.Render(w, r, forms.RecurringEventForm(formProps))
		return
	}

	w.Header().Set("HX-Redirect", routeurl.URL("page.app.spaces.space.recurring", "spaceID", spaceID))
	w.WriteHeader(http.StatusOK)
}

func (h *recurringEventHandler) HandleDelete(w http.ResponseWriter, r *http.Request) {
	spaceID := r.PathValue("spaceID")
	eventID := r.PathValue("eventID")
	existing, err := h.recurringService.Get(eventID)
	if err != nil || existing.SpaceID != spaceID {
		ui.RenderError(w, r, "Recurring event not found", http.StatusNotFound)
		return
	}
	if err := h.recurringService.Delete(eventID); err != nil {
		slog.Error("failed to delete recurring event", "error", err, "event_id", eventID)
		ui.RenderError(w, r, "Failed to delete", http.StatusInternalServerError)
		return
	}
	w.Header().Set("HX-Redirect", routeurl.URL("page.app.spaces.space.recurring", "spaceID", spaceID))
	w.WriteHeader(http.StatusOK)
}

func (h *recurringEventHandler) HandlePause(w http.ResponseWriter, r *http.Request) {
	h.setPaused(w, r, true)
}

func (h *recurringEventHandler) HandleResume(w http.ResponseWriter, r *http.Request) {
	h.setPaused(w, r, false)
}

func (h *recurringEventHandler) setPaused(w http.ResponseWriter, r *http.Request, paused bool) {
	spaceID := r.PathValue("spaceID")
	eventID := r.PathValue("eventID")
	existing, err := h.recurringService.Get(eventID)
	if err != nil || existing.SpaceID != spaceID {
		ui.RenderError(w, r, "Recurring event not found", http.StatusNotFound)
		return
	}
	if err := h.recurringService.SetPaused(eventID, paused); err != nil {
		slog.Error("failed to toggle pause", "error", err, "event_id", eventID)
		ui.RenderError(w, r, "Failed to update", http.StatusInternalServerError)
		return
	}
	w.Header().Set("HX-Redirect", routeurl.URL("page.app.spaces.space.recurring", "spaceID", spaceID))
	w.WriteHeader(http.StatusOK)
}

// parseForm reads the recurring-event form, returns a populated CreateRecurringEventInput
// alongside form props echoed back to the user with field-level errors.
func (h *recurringEventHandler) parseForm(r *http.Request, spaceID string) (service.CreateRecurringEventInput, forms.RecurringEventFormProps) {
	accounts, _ := h.accountService.GetAccountsForSpace(spaceID)

	title := strings.TrimSpace(r.FormValue("title"))
	kind := strings.TrimSpace(r.FormValue("kind"))
	sourceID := strings.TrimSpace(r.FormValue("source_account"))
	amountStr := strings.TrimSpace(r.FormValue("amount"))
	descriptionStr := strings.TrimSpace(r.FormValue("description"))
	frequency := strings.TrimSpace(r.FormValue("frequency"))
	intervalStr := strings.TrimSpace(r.FormValue("interval_count"))
	dowStr := strings.TrimSpace(r.FormValue("day_of_week"))
	domStr := strings.TrimSpace(r.FormValue("day_of_month"))
	moyStr := strings.TrimSpace(r.FormValue("month_of_year"))
	fireTime := strings.TrimSpace(r.FormValue("fire_time"))
	tz := strings.TrimSpace(r.FormValue("timezone"))
	startDateStr := strings.TrimSpace(r.FormValue("start_date"))

	props := forms.RecurringEventFormProps{
		SpaceID:         spaceID,
		Accounts:        accounts,
		Timezones:       timezone.CommonTimezones(),
		Title:           title,
		Kind:            kind,
		SourceAccountID: sourceID,
		Amount:          amountStr,
		Description:     descriptionStr,
		Frequency:       frequency,
		IntervalCount:   intervalStr,
		DayOfWeek:       dowStr,
		DayOfMonth:      domStr,
		MonthOfYear:     moyStr,
		FireTime:        fireTime,
		Timezone:        tz,
		StartDate:       startDateStr,
	}

	input := service.CreateRecurringEventInput{
		SpaceID:         spaceID,
		Kind:            model.RecurringEventKind(kind),
		SourceAccountID: sourceID,
		Title:           title,
		Description:     descriptionStr,
		Frequency:       model.RecurringFrequency(frequency),
		Timezone:        tz,
	}

	if title == "" {
		props.TitleErr = "Title is required."
	}
	switch model.RecurringEventKind(kind) {
	case model.RecurringEventKindBill, model.RecurringEventKindFund:
		// ok
	default:
		props.KindErr = "Choose a kind."
	}
	if sourceID == "" {
		props.SourceErr = "Source account is required."
	}
	if amount, err := decimal.NewFromString(amountStr); err != nil {
		props.AmountErr = "Enter a valid amount (e.g. 12.34)."
	} else if !amount.IsPositive() {
		props.AmountErr = "Amount must be greater than zero."
	} else if amount.Exponent() < -2 {
		props.AmountErr = "Amount can have at most 2 decimal places."
	} else {
		input.Amount = amount
	}

	if interval, err := strconv.Atoi(intervalStr); err != nil || interval < 1 {
		props.IntervalErr = "Interval must be a positive whole number."
	} else {
		input.IntervalCount = interval
	}

	switch model.RecurringFrequency(frequency) {
	case model.RecurringFrequencyDaily:
		// no anchor
	case model.RecurringFrequencyWeekly:
		if v, err := strconv.Atoi(dowStr); err != nil || v < 0 || v > 6 {
			props.DayOfWeekErr = "Choose a day of the week."
		} else {
			input.DayOfWeek = &v
		}
	case model.RecurringFrequencyMonthly:
		if v, err := strconv.Atoi(domStr); err != nil || v < 1 || v > 31 {
			props.DayOfMonthErr = "Day of month must be between 1 and 31."
		} else {
			input.DayOfMonth = &v
		}
	case model.RecurringFrequencyYearly:
		if v, err := strconv.Atoi(domStr); err != nil || v < 1 || v > 31 {
			props.DayOfMonthErr = "Day of month must be between 1 and 31."
		} else {
			input.DayOfMonth = &v
		}
		if v, err := strconv.Atoi(moyStr); err != nil || v < 1 || v > 12 {
			props.MonthOfYearErr = "Month must be between 1 and 12."
		} else {
			input.MonthOfYear = &v
		}
	default:
		props.FrequencyErr = "Choose a frequency."
	}

	if hh, mm, ok := parseTimeOfDay(fireTime); !ok {
		props.FireTimeErr = "Enter a valid time (HH:MM)."
	} else {
		input.FireHour = hh
		input.FireMinute = mm
	}

	if tz == "" {
		props.TimezoneErr = "Timezone is required."
	} else if _, err := time.LoadLocation(tz); err != nil {
		props.TimezoneErr = "Unknown timezone."
	}

	if startDateStr == "" {
		props.StartDateErr = "Start date is required."
	} else if d, err := time.Parse("2006-01-02", startDateStr); err != nil {
		props.StartDateErr = "Enter a valid date."
	} else {
		input.StartDate = d
	}

	return input, props
}

func parseTimeOfDay(s string) (int, int, bool) {
	parts := strings.Split(s, ":")
	if len(parts) != 2 {
		return 0, 0, false
	}
	h, err1 := strconv.Atoi(parts[0])
	m, err2 := strconv.Atoi(parts[1])
	if err1 != nil || err2 != nil || h < 0 || h > 23 || m < 0 || m > 59 {
		return 0, 0, false
	}
	return h, m, true
}

func formatTimeOfDay(h, m int) string {
	return strconv.Itoa(h) + ":" + leadingZero(m)
}

func leadingZero(n int) string {
	if n < 10 {
		return "0" + strconv.Itoa(n)
	}
	return strconv.Itoa(n)
}

func mustLoc(name string) *time.Location {
	loc, err := time.LoadLocation(name)
	if err != nil {
		return time.UTC
	}
	return loc
}

func friendlyRecurringError(err error) string {
	if err == nil {
		return ""
	}
	if errors.Is(err, repository.ErrRecurringEventNotFound) {
		return "Recurring event not found."
	}
	return err.Error()
}
