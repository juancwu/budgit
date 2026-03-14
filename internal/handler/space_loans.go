package handler

import (
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/shopspring/decimal"

	"git.juancwu.dev/juancwu/budgit/internal/ctxkeys"
	"git.juancwu.dev/juancwu/budgit/internal/model"
	"git.juancwu.dev/juancwu/budgit/internal/service"
	"git.juancwu.dev/juancwu/budgit/internal/ui"
	"git.juancwu.dev/juancwu/budgit/internal/ui/pages"
)

func (h *SpaceHandler) LoansPage(w http.ResponseWriter, r *http.Request) {
	spaceID := r.PathValue("spaceID")
	space, err := h.spaceService.GetSpace(spaceID)
	if err != nil {
		slog.Error("failed to get space", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	pageStr := r.URL.Query().Get("page")
	page, _ := strconv.Atoi(pageStr)
	if page < 1 {
		page = 1
	}

	loans, totalPages, err := h.loanService.GetLoansWithSummaryForSpacePaginated(spaceID, page)
	if err != nil {
		slog.Error("failed to get loans", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	ui.Render(w, r, pages.SpaceLoansPage(space, loans, page, totalPages))
}

func (h *SpaceHandler) CreateLoan(w http.ResponseWriter, r *http.Request) {
	spaceID := r.PathValue("spaceID")
	user := ctxkeys.User(r.Context())

	name := strings.TrimSpace(r.FormValue("name"))
	if name == "" {
		w.Header().Set("HX-Reswap", "none")
		w.WriteHeader(http.StatusUnprocessableEntity)
		return
	}

	description := strings.TrimSpace(r.FormValue("description"))

	amountStr := r.FormValue("amount")
	amount, err := decimal.NewFromString(amountStr)
	if err != nil || amount.LessThanOrEqual(decimal.Zero) {
		w.Header().Set("HX-Reswap", "none")
		w.WriteHeader(http.StatusUnprocessableEntity)
		return
	}
	amountCents := int(amount.Mul(decimal.NewFromInt(100)).IntPart())

	interestStr := r.FormValue("interest_rate")
	var interestBps int
	if interestStr != "" {
		interestRate, err := decimal.NewFromString(interestStr)
		if err == nil {
			interestBps = int(interestRate.Mul(decimal.NewFromInt(100)).IntPart())
		}
	}

	startDateStr := r.FormValue("start_date")
	startDate, err := time.Parse("2006-01-02", startDateStr)
	if err != nil {
		startDate = time.Now()
	}

	var endDate *time.Time
	endDateStr := r.FormValue("end_date")
	if endDateStr != "" {
		parsed, err := time.Parse("2006-01-02", endDateStr)
		if err == nil {
			endDate = &parsed
		}
	}

	dto := service.CreateLoanDTO{
		SpaceID:         spaceID,
		UserID:          user.ID,
		Name:            name,
		Description:     description,
		OriginalAmount:  amountCents,
		InterestRateBps: interestBps,
		StartDate:       startDate,
		EndDate:         endDate,
	}

	_, err = h.loanService.CreateLoan(dto)
	if err != nil {
		slog.Error("failed to create loan", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Return updated loans list
	loans, totalPages, err := h.loanService.GetLoansWithSummaryForSpacePaginated(spaceID, 1)
	if err != nil {
		slog.Error("failed to get loans after create", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	ui.Render(w, r, pages.LoansListContent(spaceID, loans, 1, totalPages))
}

func (h *SpaceHandler) LoanDetailPage(w http.ResponseWriter, r *http.Request) {
	spaceID := r.PathValue("spaceID")
	loanID := r.PathValue("loanID")

	space, err := h.spaceService.GetSpace(spaceID)
	if err != nil {
		slog.Error("failed to get space", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	loan, err := h.loanService.GetLoanWithSummary(loanID)
	if err != nil {
		slog.Error("failed to get loan", "error", err)
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}

	pageStr := r.URL.Query().Get("page")
	page, _ := strconv.Atoi(pageStr)
	if page < 1 {
		page = 1
	}

	receipts, totalPages, err := h.receiptService.GetReceiptsForLoanPaginated(loanID, page)
	if err != nil {
		slog.Error("failed to get receipts", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	recurringReceipts, err := h.recurringReceiptService.GetRecurringReceiptsWithSourcesForLoan(loanID)
	if err != nil {
		slog.Error("failed to get recurring receipts", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	accounts, err := h.accountService.GetAccountsForSpace(spaceID)
	if err != nil {
		slog.Error("failed to get accounts", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	balance, err := h.expenseService.GetBalanceForSpace(spaceID)
	if err != nil {
		slog.Error("failed to get balance", "error", err)
		balance = 0
	}

	ui.Render(w, r, pages.SpaceLoanDetailPage(space, loan, receipts, page, totalPages, recurringReceipts, accounts, balance))
}

func (h *SpaceHandler) UpdateLoan(w http.ResponseWriter, r *http.Request) {
	loanID := r.PathValue("loanID")

	name := strings.TrimSpace(r.FormValue("name"))
	if name == "" {
		w.Header().Set("HX-Reswap", "none")
		w.WriteHeader(http.StatusUnprocessableEntity)
		return
	}

	description := strings.TrimSpace(r.FormValue("description"))

	amountStr := r.FormValue("amount")
	amount, err := decimal.NewFromString(amountStr)
	if err != nil || amount.LessThanOrEqual(decimal.Zero) {
		w.Header().Set("HX-Reswap", "none")
		w.WriteHeader(http.StatusUnprocessableEntity)
		return
	}
	amountCents := int(amount.Mul(decimal.NewFromInt(100)).IntPart())

	interestStr := r.FormValue("interest_rate")
	var interestBps int
	if interestStr != "" {
		interestRate, err := decimal.NewFromString(interestStr)
		if err == nil {
			interestBps = int(interestRate.Mul(decimal.NewFromInt(100)).IntPart())
		}
	}

	startDateStr := r.FormValue("start_date")
	startDate, err := time.Parse("2006-01-02", startDateStr)
	if err != nil {
		startDate = time.Now()
	}

	var endDate *time.Time
	endDateStr := r.FormValue("end_date")
	if endDateStr != "" {
		parsed, err := time.Parse("2006-01-02", endDateStr)
		if err == nil {
			endDate = &parsed
		}
	}

	dto := service.UpdateLoanDTO{
		ID:              loanID,
		Name:            name,
		Description:     description,
		OriginalAmount:  amountCents,
		InterestRateBps: interestBps,
		StartDate:       startDate,
		EndDate:         endDate,
	}

	_, err = h.loanService.UpdateLoan(dto)
	if err != nil {
		slog.Error("failed to update loan", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Redirect to loan detail
	spaceID := r.PathValue("spaceID")
	w.Header().Set("HX-Redirect", fmt.Sprintf("/app/spaces/%s/loans/%s", spaceID, loanID))
	w.WriteHeader(http.StatusOK)
}

func (h *SpaceHandler) DeleteLoan(w http.ResponseWriter, r *http.Request) {
	spaceID := r.PathValue("spaceID")
	loanID := r.PathValue("loanID")

	if err := h.loanService.DeleteLoan(loanID); err != nil {
		slog.Error("failed to delete loan", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("HX-Redirect", fmt.Sprintf("/app/spaces/%s/loans", spaceID))
	w.WriteHeader(http.StatusOK)
}

func (h *SpaceHandler) CreateReceipt(w http.ResponseWriter, r *http.Request) {
	spaceID := r.PathValue("spaceID")
	loanID := r.PathValue("loanID")
	user := ctxkeys.User(r.Context())

	description := strings.TrimSpace(r.FormValue("description"))

	amountStr := r.FormValue("amount")
	amount, err := decimal.NewFromString(amountStr)
	if err != nil || amount.LessThanOrEqual(decimal.Zero) {
		w.Header().Set("HX-Reswap", "none")
		w.WriteHeader(http.StatusUnprocessableEntity)
		return
	}
	totalAmountCents := int(amount.Mul(decimal.NewFromInt(100)).IntPart())

	dateStr := r.FormValue("date")
	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		date = time.Now()
	}

	// Parse funding sources from parallel arrays
	fundingSources, err := parseFundingSources(r)
	if err != nil {
		w.Header().Set("HX-Reswap", "none")
		w.WriteHeader(http.StatusUnprocessableEntity)
		return
	}

	dto := service.CreateReceiptDTO{
		LoanID:         loanID,
		SpaceID:        spaceID,
		UserID:         user.ID,
		Description:    description,
		TotalAmount:    totalAmountCents,
		Date:           date,
		FundingSources: fundingSources,
	}

	_, err = h.receiptService.CreateReceipt(dto)
	if err != nil {
		slog.Error("failed to create receipt", "error", err)
		ui.RenderError(w, r, err.Error(), http.StatusUnprocessableEntity)
		return
	}

	// Return updated loan detail
	w.Header().Set("HX-Redirect", fmt.Sprintf("/app/spaces/%s/loans/%s", spaceID, loanID))
	w.WriteHeader(http.StatusOK)
}

func (h *SpaceHandler) UpdateReceipt(w http.ResponseWriter, r *http.Request) {
	spaceID := r.PathValue("spaceID")
	loanID := r.PathValue("loanID")
	receiptID := r.PathValue("receiptID")
	user := ctxkeys.User(r.Context())

	description := strings.TrimSpace(r.FormValue("description"))

	amountStr := r.FormValue("amount")
	amount, err := decimal.NewFromString(amountStr)
	if err != nil || amount.LessThanOrEqual(decimal.Zero) {
		w.Header().Set("HX-Reswap", "none")
		w.WriteHeader(http.StatusUnprocessableEntity)
		return
	}
	totalAmountCents := int(amount.Mul(decimal.NewFromInt(100)).IntPart())

	dateStr := r.FormValue("date")
	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		date = time.Now()
	}

	fundingSources, err := parseFundingSources(r)
	if err != nil {
		w.Header().Set("HX-Reswap", "none")
		w.WriteHeader(http.StatusUnprocessableEntity)
		return
	}

	dto := service.UpdateReceiptDTO{
		ID:             receiptID,
		SpaceID:        spaceID,
		UserID:         user.ID,
		Description:    description,
		TotalAmount:    totalAmountCents,
		Date:           date,
		FundingSources: fundingSources,
	}

	_, err = h.receiptService.UpdateReceipt(dto)
	if err != nil {
		slog.Error("failed to update receipt", "error", err)
		ui.RenderError(w, r, err.Error(), http.StatusUnprocessableEntity)
		return
	}

	w.Header().Set("HX-Redirect", fmt.Sprintf("/app/spaces/%s/loans/%s", spaceID, loanID))
	w.WriteHeader(http.StatusOK)
}

func (h *SpaceHandler) DeleteReceipt(w http.ResponseWriter, r *http.Request) {
	spaceID := r.PathValue("spaceID")
	loanID := r.PathValue("loanID")
	receiptID := r.PathValue("receiptID")

	if err := h.receiptService.DeleteReceipt(receiptID, spaceID); err != nil {
		slog.Error("failed to delete receipt", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("HX-Redirect", fmt.Sprintf("/app/spaces/%s/loans/%s", spaceID, loanID))
	w.WriteHeader(http.StatusOK)
}

func (h *SpaceHandler) GetReceiptsList(w http.ResponseWriter, r *http.Request) {
	spaceID := r.PathValue("spaceID")
	loanID := r.PathValue("loanID")

	pageStr := r.URL.Query().Get("page")
	page, _ := strconv.Atoi(pageStr)
	if page < 1 {
		page = 1
	}

	receipts, totalPages, err := h.receiptService.GetReceiptsForLoanPaginated(loanID, page)
	if err != nil {
		slog.Error("failed to get receipts", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	ui.Render(w, r, pages.ReceiptsListContent(spaceID, loanID, receipts, page, totalPages))
}

func (h *SpaceHandler) CreateRecurringReceipt(w http.ResponseWriter, r *http.Request) {
	spaceID := r.PathValue("spaceID")
	loanID := r.PathValue("loanID")
	user := ctxkeys.User(r.Context())

	description := strings.TrimSpace(r.FormValue("description"))

	amountStr := r.FormValue("amount")
	amount, err := decimal.NewFromString(amountStr)
	if err != nil || amount.LessThanOrEqual(decimal.Zero) {
		w.Header().Set("HX-Reswap", "none")
		w.WriteHeader(http.StatusUnprocessableEntity)
		return
	}
	totalAmountCents := int(amount.Mul(decimal.NewFromInt(100)).IntPart())

	frequency := model.Frequency(r.FormValue("frequency"))

	startDateStr := r.FormValue("start_date")
	startDate, err := time.Parse("2006-01-02", startDateStr)
	if err != nil {
		startDate = time.Now()
	}

	var endDate *time.Time
	endDateStr := r.FormValue("end_date")
	if endDateStr != "" {
		parsed, err := time.Parse("2006-01-02", endDateStr)
		if err == nil {
			endDate = &parsed
		}
	}

	fundingSources, err := parseFundingSources(r)
	if err != nil {
		w.Header().Set("HX-Reswap", "none")
		w.WriteHeader(http.StatusUnprocessableEntity)
		return
	}

	dto := service.CreateRecurringReceiptDTO{
		LoanID:         loanID,
		SpaceID:        spaceID,
		UserID:         user.ID,
		Description:    description,
		TotalAmount:    totalAmountCents,
		Frequency:      frequency,
		StartDate:      startDate,
		EndDate:        endDate,
		FundingSources: fundingSources,
	}

	_, err = h.recurringReceiptService.CreateRecurringReceipt(dto)
	if err != nil {
		slog.Error("failed to create recurring receipt", "error", err)
		ui.RenderError(w, r, err.Error(), http.StatusUnprocessableEntity)
		return
	}

	w.Header().Set("HX-Redirect", fmt.Sprintf("/app/spaces/%s/loans/%s", spaceID, loanID))
	w.WriteHeader(http.StatusOK)
}

func (h *SpaceHandler) UpdateRecurringReceipt(w http.ResponseWriter, r *http.Request) {
	spaceID := r.PathValue("spaceID")
	loanID := r.PathValue("loanID")
	recurringReceiptID := r.PathValue("recurringReceiptID")

	description := strings.TrimSpace(r.FormValue("description"))

	amountStr := r.FormValue("amount")
	amount, err := decimal.NewFromString(amountStr)
	if err != nil || amount.LessThanOrEqual(decimal.Zero) {
		w.Header().Set("HX-Reswap", "none")
		w.WriteHeader(http.StatusUnprocessableEntity)
		return
	}
	totalAmountCents := int(amount.Mul(decimal.NewFromInt(100)).IntPart())

	frequency := model.Frequency(r.FormValue("frequency"))

	startDateStr := r.FormValue("start_date")
	startDate, err := time.Parse("2006-01-02", startDateStr)
	if err != nil {
		startDate = time.Now()
	}

	var endDate *time.Time
	endDateStr := r.FormValue("end_date")
	if endDateStr != "" {
		parsed, err := time.Parse("2006-01-02", endDateStr)
		if err == nil {
			endDate = &parsed
		}
	}

	fundingSources, err := parseFundingSources(r)
	if err != nil {
		w.Header().Set("HX-Reswap", "none")
		w.WriteHeader(http.StatusUnprocessableEntity)
		return
	}

	dto := service.UpdateRecurringReceiptDTO{
		ID:             recurringReceiptID,
		Description:    description,
		TotalAmount:    totalAmountCents,
		Frequency:      frequency,
		StartDate:      startDate,
		EndDate:        endDate,
		FundingSources: fundingSources,
	}

	_, err = h.recurringReceiptService.UpdateRecurringReceipt(dto)
	if err != nil {
		slog.Error("failed to update recurring receipt", "error", err)
		ui.RenderError(w, r, err.Error(), http.StatusUnprocessableEntity)
		return
	}

	w.Header().Set("HX-Redirect", fmt.Sprintf("/app/spaces/%s/loans/%s", spaceID, loanID))
	w.WriteHeader(http.StatusOK)
}

func (h *SpaceHandler) DeleteRecurringReceipt(w http.ResponseWriter, r *http.Request) {
	spaceID := r.PathValue("spaceID")
	loanID := r.PathValue("loanID")
	recurringReceiptID := r.PathValue("recurringReceiptID")

	if err := h.recurringReceiptService.DeleteRecurringReceipt(recurringReceiptID); err != nil {
		slog.Error("failed to delete recurring receipt", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("HX-Redirect", fmt.Sprintf("/app/spaces/%s/loans/%s", spaceID, loanID))
	w.WriteHeader(http.StatusOK)
}

func (h *SpaceHandler) ToggleRecurringReceipt(w http.ResponseWriter, r *http.Request) {
	spaceID := r.PathValue("spaceID")
	loanID := r.PathValue("loanID")
	recurringReceiptID := r.PathValue("recurringReceiptID")

	_, err := h.recurringReceiptService.ToggleRecurringReceipt(recurringReceiptID)
	if err != nil {
		slog.Error("failed to toggle recurring receipt", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("HX-Redirect", fmt.Sprintf("/app/spaces/%s/loans/%s", spaceID, loanID))
	w.WriteHeader(http.StatusOK)
}

// parseFundingSources parses funding sources from parallel form arrays:
// source_type[], source_amount[], source_account_id[]
func parseFundingSources(r *http.Request) ([]service.FundingSourceDTO, error) {
	if err := r.ParseForm(); err != nil {
		return nil, err
	}

	sourceTypes := r.Form["source_type"]
	sourceAmounts := r.Form["source_amount"]
	sourceAccountIDs := r.Form["source_account_id"]

	if len(sourceTypes) == 0 {
		return nil, fmt.Errorf("no funding sources provided")
	}
	if len(sourceTypes) != len(sourceAmounts) {
		return nil, fmt.Errorf("mismatched funding source fields")
	}

	var sources []service.FundingSourceDTO
	for i, srcType := range sourceTypes {
		amount, err := decimal.NewFromString(sourceAmounts[i])
		if err != nil || amount.LessThanOrEqual(decimal.Zero) {
			return nil, fmt.Errorf("invalid funding source amount")
		}
		amountCents := int(amount.Mul(decimal.NewFromInt(100)).IntPart())

		src := service.FundingSourceDTO{
			SourceType: model.FundingSourceType(srcType),
			Amount:     amountCents,
		}

		if srcType == string(model.FundingSourceAccount) {
			if i < len(sourceAccountIDs) && sourceAccountIDs[i] != "" {
				src.AccountID = sourceAccountIDs[i]
			} else {
				return nil, fmt.Errorf("account source requires account_id")
			}
		}

		sources = append(sources, src)
	}

	return sources, nil
}
