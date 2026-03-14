package handler

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/shopspring/decimal"

	"git.juancwu.dev/juancwu/budgit/internal/ctxkeys"
	"git.juancwu.dev/juancwu/budgit/internal/model"
	"git.juancwu.dev/juancwu/budgit/internal/service"
	"git.juancwu.dev/juancwu/budgit/internal/ui"
	"git.juancwu.dev/juancwu/budgit/internal/ui/components/recurring"
	"git.juancwu.dev/juancwu/budgit/internal/ui/components/toast"
	"git.juancwu.dev/juancwu/budgit/internal/ui/pages"
)

type RecurringHandler struct {
	spaceService     *service.SpaceService
	recurringService *service.RecurringExpenseService
	tagService       *service.TagService
	methodService    *service.PaymentMethodService
}

func NewRecurringHandler(ss *service.SpaceService, rs *service.RecurringExpenseService, ts *service.TagService, pms *service.PaymentMethodService) *RecurringHandler {
	return &RecurringHandler{
		spaceService:     ss,
		recurringService: rs,
		tagService:       ts,
		methodService:    pms,
	}
}

func (h *RecurringHandler) getRecurringForSpace(w http.ResponseWriter, spaceID, recurringID string) *model.RecurringExpense {
	re, err := h.recurringService.GetRecurringExpense(recurringID)
	if err != nil {
		http.Error(w, "Recurring expense not found", http.StatusNotFound)
		return nil
	}
	if re.SpaceID != spaceID {
		http.Error(w, "Not Found", http.StatusNotFound)
		return nil
	}
	return re
}

func (h *RecurringHandler) RecurringExpensesPage(w http.ResponseWriter, r *http.Request) {
	spaceID := r.PathValue("spaceID")
	space, err := h.spaceService.GetSpace(spaceID)
	if err != nil {
		http.Error(w, "Space not found", http.StatusNotFound)
		return
	}

	// Lazy check: process any due recurrences for this space
	h.recurringService.ProcessDueRecurrencesForSpace(spaceID, time.Now())

	recs, err := h.recurringService.GetRecurringExpensesWithTagsAndMethodsForSpace(spaceID)
	if err != nil {
		slog.Error("failed to get recurring expenses", "error", err, "space_id", spaceID)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	tags, err := h.tagService.GetTagsForSpace(spaceID)
	if err != nil {
		slog.Error("failed to get tags", "error", err, "space_id", spaceID)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	methods, err := h.methodService.GetMethodsForSpace(spaceID)
	if err != nil {
		slog.Error("failed to get payment methods", "error", err, "space_id", spaceID)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	ui.Render(w, r, pages.SpaceRecurringPage(space, recs, tags, methods))
}

func (h *RecurringHandler) CreateRecurringExpense(w http.ResponseWriter, r *http.Request) {
	spaceID := r.PathValue("spaceID")
	user := ctxkeys.User(r.Context())

	if err := r.ParseForm(); err != nil {
		ui.RenderError(w, r, "Bad Request", http.StatusUnprocessableEntity)
		return
	}

	description := r.FormValue("description")
	amountStr := r.FormValue("amount")
	typeStr := r.FormValue("type")
	frequencyStr := r.FormValue("frequency")
	startDateStr := r.FormValue("start_date")
	endDateStr := r.FormValue("end_date")
	tagNames := r.Form["tags"]

	if description == "" || amountStr == "" || typeStr == "" || frequencyStr == "" || startDateStr == "" {
		ui.RenderError(w, r, "All required fields must be provided.", http.StatusUnprocessableEntity)
		return
	}

	amountDecimal, err := decimal.NewFromString(amountStr)
	if err != nil {
		ui.RenderError(w, r, "Invalid amount format.", http.StatusUnprocessableEntity)
		return
	}
	amountCents := int(amountDecimal.Mul(decimal.NewFromInt(100)).IntPart())

	startDate, err := time.Parse("2006-01-02", startDateStr)
	if err != nil {
		ui.RenderError(w, r, "Invalid start date format.", http.StatusUnprocessableEntity)
		return
	}

	var endDate *time.Time
	if endDateStr != "" {
		ed, err := time.Parse("2006-01-02", endDateStr)
		if err != nil {
			ui.RenderError(w, r, "Invalid end date format.", http.StatusUnprocessableEntity)
			return
		}
		endDate = &ed
	}

	expenseType := model.ExpenseType(typeStr)
	if expenseType != model.ExpenseTypeExpense && expenseType != model.ExpenseTypeTopup {
		ui.RenderError(w, r, "Invalid transaction type.", http.StatusUnprocessableEntity)
		return
	}

	frequency := model.Frequency(frequencyStr)

	// Tag processing
	existingTags, err := h.tagService.GetTagsForSpace(spaceID)
	if err != nil {
		slog.Error("failed to get tags", "error", err, "space_id", spaceID)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	existingTagsMap := make(map[string]string)
	for _, t := range existingTags {
		existingTagsMap[t.Name] = t.ID
	}

	var finalTagIDs []string
	processedTags := make(map[string]bool)
	for _, rawTagName := range tagNames {
		tagName := service.NormalizeTagName(rawTagName)
		if tagName == "" || processedTags[tagName] {
			continue
		}
		if id, exists := existingTagsMap[tagName]; exists {
			finalTagIDs = append(finalTagIDs, id)
		} else {
			newTag, err := h.tagService.CreateTag(spaceID, tagName, nil)
			if err != nil {
				slog.Error("failed to create tag", "error", err, "tag_name", tagName)
				continue
			}
			finalTagIDs = append(finalTagIDs, newTag.ID)
			existingTagsMap[tagName] = newTag.ID
		}
		processedTags[tagName] = true
	}

	var paymentMethodID *string
	if pmid := r.FormValue("payment_method_id"); pmid != "" {
		paymentMethodID = &pmid
	}

	re, err := h.recurringService.CreateRecurringExpense(service.CreateRecurringExpenseDTO{
		SpaceID:         spaceID,
		UserID:          user.ID,
		Description:     description,
		Amount:          amountCents,
		Type:            expenseType,
		PaymentMethodID: paymentMethodID,
		Frequency:       frequency,
		StartDate:       startDate,
		EndDate:         endDate,
		TagIDs:          finalTagIDs,
	})
	if err != nil {
		slog.Error("failed to create recurring expense", "error", err)
		http.Error(w, "Failed to create recurring expense.", http.StatusInternalServerError)
		return
	}

	// Fetch tags/method for the response
	spaceTags, _ := h.tagService.GetTagsForSpace(spaceID)
	tagsMap, _ := h.recurringService.GetRecurringExpensesWithTagsAndMethodsForSpace(spaceID)
	for _, item := range tagsMap {
		if item.ID == re.ID {
			ui.Render(w, r, recurring.RecurringItem(spaceID, item, nil, spaceTags))
			return
		}
	}

	// Fallback: render without tags
	ui.Render(w, r, recurring.RecurringItem(spaceID, &model.RecurringExpenseWithTagsAndMethod{RecurringExpense: *re}, nil, spaceTags))
}

func (h *RecurringHandler) UpdateRecurringExpense(w http.ResponseWriter, r *http.Request) {
	spaceID := r.PathValue("spaceID")
	recurringID := r.PathValue("recurringID")

	if h.getRecurringForSpace(w, spaceID, recurringID) == nil {
		return
	}

	if err := r.ParseForm(); err != nil {
		ui.RenderError(w, r, "Bad Request", http.StatusUnprocessableEntity)
		return
	}

	description := r.FormValue("description")
	amountStr := r.FormValue("amount")
	typeStr := r.FormValue("type")
	frequencyStr := r.FormValue("frequency")
	startDateStr := r.FormValue("start_date")
	endDateStr := r.FormValue("end_date")
	tagNames := r.Form["tags"]

	if description == "" || amountStr == "" || typeStr == "" || frequencyStr == "" || startDateStr == "" {
		ui.RenderError(w, r, "All required fields must be provided.", http.StatusUnprocessableEntity)
		return
	}

	amountDecimal, err := decimal.NewFromString(amountStr)
	if err != nil {
		ui.RenderError(w, r, "Invalid amount.", http.StatusUnprocessableEntity)
		return
	}
	amountCents := int(amountDecimal.Mul(decimal.NewFromInt(100)).IntPart())

	startDate, err := time.Parse("2006-01-02", startDateStr)
	if err != nil {
		ui.RenderError(w, r, "Invalid start date.", http.StatusUnprocessableEntity)
		return
	}

	var endDate *time.Time
	if endDateStr != "" {
		ed, err := time.Parse("2006-01-02", endDateStr)
		if err != nil {
			ui.RenderError(w, r, "Invalid end date.", http.StatusUnprocessableEntity)
			return
		}
		endDate = &ed
	}

	// Tag processing
	existingTags, err := h.tagService.GetTagsForSpace(spaceID)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	existingTagsMap := make(map[string]string)
	for _, t := range existingTags {
		existingTagsMap[t.Name] = t.ID
	}
	var finalTagIDs []string
	processedTags := make(map[string]bool)
	for _, rawTagName := range tagNames {
		tagName := service.NormalizeTagName(rawTagName)
		if tagName == "" || processedTags[tagName] {
			continue
		}
		if id, exists := existingTagsMap[tagName]; exists {
			finalTagIDs = append(finalTagIDs, id)
		} else {
			newTag, err := h.tagService.CreateTag(spaceID, tagName, nil)
			if err != nil {
				continue
			}
			finalTagIDs = append(finalTagIDs, newTag.ID)
		}
		processedTags[tagName] = true
	}

	var paymentMethodID *string
	if pmid := r.FormValue("payment_method_id"); pmid != "" {
		paymentMethodID = &pmid
	}

	updated, err := h.recurringService.UpdateRecurringExpense(service.UpdateRecurringExpenseDTO{
		ID:              recurringID,
		Description:     description,
		Amount:          amountCents,
		Type:            model.ExpenseType(typeStr),
		PaymentMethodID: paymentMethodID,
		Frequency:       model.Frequency(frequencyStr),
		StartDate:       startDate,
		EndDate:         endDate,
		TagIDs:          finalTagIDs,
	})
	if err != nil {
		slog.Error("failed to update recurring expense", "error", err)
		http.Error(w, "Failed to update.", http.StatusInternalServerError)
		return
	}

	// Build response with tags/method
	updateSpaceTags, _ := h.tagService.GetTagsForSpace(spaceID)
	tagsMapResult, _ := h.recurringService.GetRecurringExpensesWithTagsAndMethodsForSpace(spaceID)
	for _, item := range tagsMapResult {
		if item.ID == updated.ID {
			methods, _ := h.methodService.GetMethodsForSpace(spaceID)
			ui.Render(w, r, recurring.RecurringItem(spaceID, item, methods, updateSpaceTags))
			return
		}
	}

	ui.Render(w, r, recurring.RecurringItem(spaceID, &model.RecurringExpenseWithTagsAndMethod{RecurringExpense: *updated}, nil, updateSpaceTags))
}

func (h *RecurringHandler) DeleteRecurringExpense(w http.ResponseWriter, r *http.Request) {
	spaceID := r.PathValue("spaceID")
	recurringID := r.PathValue("recurringID")

	if h.getRecurringForSpace(w, spaceID, recurringID) == nil {
		return
	}

	if err := h.recurringService.DeleteRecurringExpense(recurringID); err != nil {
		slog.Error("failed to delete recurring expense", "error", err, "recurring_id", recurringID)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	ui.RenderToast(w, r, toast.Toast(toast.Props{
		Title:       "Recurring expense deleted",
		Variant:     toast.VariantSuccess,
		Icon:        true,
		Dismissible: true,
		Duration:    5000,
	}))
}

func (h *RecurringHandler) ToggleRecurringExpense(w http.ResponseWriter, r *http.Request) {
	spaceID := r.PathValue("spaceID")
	recurringID := r.PathValue("recurringID")

	if h.getRecurringForSpace(w, spaceID, recurringID) == nil {
		return
	}

	updated, err := h.recurringService.ToggleRecurringExpense(recurringID)
	if err != nil {
		slog.Error("failed to toggle recurring expense", "error", err, "recurring_id", recurringID)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	toggleSpaceTags, _ := h.tagService.GetTagsForSpace(spaceID)
	tagsMapResult, _ := h.recurringService.GetRecurringExpensesWithTagsAndMethodsForSpace(spaceID)
	for _, item := range tagsMapResult {
		if item.ID == updated.ID {
			methods, _ := h.methodService.GetMethodsForSpace(spaceID)
			ui.Render(w, r, recurring.RecurringItem(spaceID, item, methods, toggleSpaceTags))
			return
		}
	}

	ui.Render(w, r, recurring.RecurringItem(spaceID, &model.RecurringExpenseWithTagsAndMethod{RecurringExpense: *updated}, nil, toggleSpaceTags))
}
