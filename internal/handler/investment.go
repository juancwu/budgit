package handler

import (
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/shopspring/decimal"

	"git.juancwu.dev/juancwu/budgit/internal/ctxkeys"
	"git.juancwu.dev/juancwu/budgit/internal/model"
	"git.juancwu.dev/juancwu/budgit/internal/routeurl"
	"git.juancwu.dev/juancwu/budgit/internal/service"
	"git.juancwu.dev/juancwu/budgit/internal/ui"
	"git.juancwu.dev/juancwu/budgit/internal/ui/blocks"
	"git.juancwu.dev/juancwu/budgit/internal/ui/pages"
)

type investmentHandler struct {
	accountService    *service.AccountService
	spaceService      *service.SpaceService
	investmentService *service.InvestmentService
}

func NewInvestmentHandler(
	accountService *service.AccountService,
	spaceService *service.SpaceService,
	investmentService *service.InvestmentService,
) *investmentHandler {
	return &investmentHandler{
		accountService:    accountService,
		spaceService:      spaceService,
		investmentService: investmentService,
	}
}

func (h *investmentHandler) loadInvestmentAccount(w http.ResponseWriter, r *http.Request) (*model.Account, bool) {
	spaceID := r.PathValue("spaceID")
	accountID := r.PathValue("accountID")
	account, err := h.accountService.GetAccount(accountID)
	if err != nil || account.SpaceID != spaceID {
		ui.Render(w, r, pages.NotFound())
		return nil, false
	}
	if !account.IsInvestment {
		ui.Render(w, r, pages.NotFound())
		return nil, false
	}
	return account, true
}

// HandleSetContributionRoom upserts the room amount for a year, then returns
// the refreshed investment section so the page can swap it in place.
func (h *investmentHandler) HandleSetContributionRoom(w http.ResponseWriter, r *http.Request) {
	account, ok := h.loadInvestmentAccount(w, r)
	if !ok {
		return
	}

	year, err := strconv.Atoi(strings.TrimSpace(r.FormValue("year")))
	if err != nil || year < 1900 || year > 9999 {
		http.Error(w, "invalid year", http.StatusBadRequest)
		return
	}
	roomStr := strings.TrimSpace(r.FormValue("room"))
	room, err := decimal.NewFromString(roomStr)
	if err != nil || room.IsNegative() {
		http.Error(w, "invalid room amount", http.StatusBadRequest)
		return
	}
	if err := h.investmentService.SetContributionRoom(account.ID, year, room); err != nil {
		slog.Error("failed to set contribution room", "error", err, "account_id", account.ID)
		http.Error(w, "could not save room", http.StatusInternalServerError)
		return
	}
	h.renderSection(w, r, account, year)
}

func (h *investmentHandler) renderSection(w http.ResponseWriter, r *http.Request, account *model.Account, year int) {
	summary, err := h.investmentService.SummarizeAccount(account.ID, year)
	if err != nil {
		slog.Error("failed to summarize investment account", "error", err, "account_id", account.ID)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	positions, err := h.investmentService.HoldingPositions(account.ID)
	if err != nil {
		slog.Error("failed to load positions", "error", err, "account_id", account.ID)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	ui.Render(w, r, blocks.InvestmentSection(blocks.InvestmentSectionProps{
		SpaceID:   account.SpaceID,
		AccountID: account.ID,
		Currency:  account.Currency,
		Summary:   summary,
		Positions: positions,
	}))
}

// ---------- Holdings ----------

func (h *investmentHandler) CreateHoldingPage(w http.ResponseWriter, r *http.Request) {
	account, ok := h.loadInvestmentAccount(w, r)
	if !ok {
		return
	}
	space, err := h.spaceService.GetSpace(account.SpaceID)
	if err != nil {
		ui.Render(w, r, pages.NotFound())
		return
	}
	ui.Render(w, r, pages.InvestmentHoldingFormPage(pages.InvestmentHoldingFormPageProps{
		SpaceID:     space.ID,
		SpaceName:   space.Name,
		AccountID:   account.ID,
		AccountName: account.Name,
	}))
}

func (h *investmentHandler) HandleCreateHolding(w http.ResponseWriter, r *http.Request) {
	account, ok := h.loadInvestmentAccount(w, r)
	if !ok {
		return
	}
	symbol := strings.ToUpper(strings.TrimSpace(r.FormValue("symbol")))
	displayName := strings.TrimSpace(r.FormValue("display_name"))
	if symbol == "" {
		http.Error(w, "symbol required", http.StatusBadRequest)
		return
	}
	if _, err := h.investmentService.CreateHolding(account.ID, symbol, displayName); err != nil {
		slog.Error("failed to create holding", "error", err)
		http.Error(w, "could not create holding", http.StatusInternalServerError)
		return
	}
	w.Header().Set("HX-Redirect", routeurl.URL(
		"page.app.spaces.space.accounts.account.overview",
		"spaceID", account.SpaceID, "accountID", account.ID,
	))
	w.WriteHeader(http.StatusOK)
}

func (h *investmentHandler) HoldingDetailPage(w http.ResponseWriter, r *http.Request) {
	account, ok := h.loadInvestmentAccount(w, r)
	if !ok {
		return
	}
	holdingID := r.PathValue("holdingID")
	holding, err := h.investmentService.GetHolding(holdingID)
	if err != nil || holding.AccountID != account.ID {
		ui.Render(w, r, pages.NotFound())
		return
	}
	pos, err := h.investmentService.HoldingPosition(holdingID)
	if err != nil {
		slog.Error("failed to load holding position", "error", err)
		ui.RenderError(w, r, "Failed to load holding", http.StatusInternalServerError)
		return
	}
	trades, err := h.investmentService.ListTrades(holdingID)
	if err != nil {
		slog.Error("failed to load trades", "error", err)
		trades = nil
	}
	space, err := h.spaceService.GetSpace(account.SpaceID)
	if err != nil {
		ui.Render(w, r, pages.NotFound())
		return
	}
	ui.Render(w, r, pages.InvestmentHoldingDetailPage(pages.InvestmentHoldingDetailProps{
		SpaceID:     space.ID,
		SpaceName:   space.Name,
		AccountID:   account.ID,
		AccountName: account.Name,
		Currency:    account.Currency,
		Position:    *pos,
		Trades:      trades,
	}))
}

func (h *investmentHandler) HandleDeleteHolding(w http.ResponseWriter, r *http.Request) {
	account, ok := h.loadInvestmentAccount(w, r)
	if !ok {
		return
	}
	holdingID := r.PathValue("holdingID")
	holding, err := h.investmentService.GetHolding(holdingID)
	if err != nil || holding.AccountID != account.ID {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if err := h.investmentService.DeleteHolding(holdingID); err != nil {
		slog.Error("failed to delete holding", "error", err)
		http.Error(w, "could not delete holding", http.StatusInternalServerError)
		return
	}
	w.Header().Set("HX-Redirect", routeurl.URL(
		"page.app.spaces.space.accounts.account.overview",
		"spaceID", account.SpaceID, "accountID", account.ID,
	))
	w.WriteHeader(http.StatusOK)
}

// ---------- Trades ----------

func (h *investmentHandler) HandleCreateTrade(w http.ResponseWriter, r *http.Request) {
	account, ok := h.loadInvestmentAccount(w, r)
	if !ok {
		return
	}
	holdingID := r.PathValue("holdingID")
	holding, err := h.investmentService.GetHolding(holdingID)
	if err != nil || holding.AccountID != account.ID {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	tradeType := strings.ToLower(strings.TrimSpace(r.FormValue("type")))
	if !model.IsValidInvestmentTradeType(tradeType) {
		http.Error(w, "invalid trade type", http.StatusBadRequest)
		return
	}
	qty, err := decimal.NewFromString(strings.TrimSpace(r.FormValue("quantity")))
	if err != nil || !qty.IsPositive() {
		http.Error(w, "invalid quantity", http.StatusBadRequest)
		return
	}
	price, err := decimal.NewFromString(strings.TrimSpace(r.FormValue("price")))
	if err != nil || price.IsNegative() {
		http.Error(w, "invalid price", http.StatusBadRequest)
		return
	}
	var feesPtr *decimal.Decimal
	if feesStr := strings.TrimSpace(r.FormValue("fees")); feesStr != "" {
		fees, err := decimal.NewFromString(feesStr)
		if err != nil || fees.IsNegative() {
			http.Error(w, "invalid fees", http.StatusBadRequest)
			return
		}
		feesPtr = &fees
	}
	occurredAt := time.Now()
	if dateStr := strings.TrimSpace(r.FormValue("occurred_at")); dateStr != "" {
		t, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			http.Error(w, "invalid date", http.StatusBadRequest)
			return
		}
		occurredAt = t
	}
	var notesPtr *string
	if notes := strings.TrimSpace(r.FormValue("notes")); notes != "" {
		notesPtr = &notes
	}

	if _, err := h.investmentService.RecordTrade(service.RecordTradeInput{
		HoldingID:    holdingID,
		Type:         model.InvestmentTradeType(tradeType),
		Quantity:     qty,
		PricePerUnit: price,
		Fees:         feesPtr,
		OccurredAt:   occurredAt,
		Notes:        notesPtr,
	}); err != nil {
		slog.Error("failed to record trade", "error", err)
		http.Error(w, "could not record trade", http.StatusInternalServerError)
		return
	}
	w.Header().Set("HX-Redirect", routeurl.URL(
		"page.app.spaces.space.accounts.account.investments.holdings.holding",
		"spaceID", account.SpaceID, "accountID", account.ID, "holdingID", holdingID,
	))
	w.WriteHeader(http.StatusOK)
}

func (h *investmentHandler) HandleDeleteTrade(w http.ResponseWriter, r *http.Request) {
	account, ok := h.loadInvestmentAccount(w, r)
	if !ok {
		return
	}
	holdingID := r.PathValue("holdingID")
	tradeID := r.PathValue("tradeID")
	trade, err := h.investmentService.GetTrade(tradeID)
	if err != nil || trade.HoldingID != holdingID {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	holding, err := h.investmentService.GetHolding(holdingID)
	if err != nil || holding.AccountID != account.ID {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if err := h.investmentService.DeleteTrade(tradeID); err != nil {
		slog.Error("failed to delete trade", "error", err)
		http.Error(w, "could not delete trade", http.StatusInternalServerError)
		return
	}
	w.Header().Set("HX-Redirect", routeurl.URL(
		"page.app.spaces.space.accounts.account.investments.holdings.holding",
		"spaceID", account.SpaceID, "accountID", account.ID, "holdingID", holdingID,
	))
	w.WriteHeader(http.StatusOK)
}

// ---------- Top-level /app/investments page ----------

func (h *investmentHandler) InvestmentsOverviewPage(w http.ResponseWriter, r *http.Request) {
	user := ctxkeys.User(r.Context())
	if user == nil {
		ui.Render(w, r, pages.NotFound())
		return
	}
	accounts, err := h.accountService.InvestmentAccountsForUser(user.ID)
	if err != nil {
		slog.Error("failed to list investment accounts", "error", err)
		ui.RenderError(w, r, "Failed to load investments", http.StatusInternalServerError)
		return
	}
	year := time.Now().Year()
	rows := make([]pages.InvestmentOverviewRow, 0, len(accounts))
	for _, acc := range accounts {
		summary, err := h.investmentService.SummarizeAccount(acc.ID, year)
		if err != nil {
			slog.Error("failed to summarize account", "error", err, "account_id", acc.ID)
			continue
		}
		space, err := h.spaceService.GetSpace(acc.SpaceID)
		spaceName := acc.SpaceID
		if err == nil {
			spaceName = space.Name
		}
		rows = append(rows, pages.InvestmentOverviewRow{
			SpaceID:     acc.SpaceID,
			SpaceName:   spaceName,
			AccountID:   acc.ID,
			AccountName: acc.Name,
			Currency:    acc.Currency,
			Summary:     summary,
		})
	}
	ui.Render(w, r, pages.InvestmentsOverviewPage(pages.InvestmentsOverviewProps{
		Year: year,
		Rows: rows,
	}))
}
