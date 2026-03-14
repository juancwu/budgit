package handler

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"git.juancwu.dev/juancwu/budgit/internal/repository"
	"git.juancwu.dev/juancwu/budgit/internal/service"
	"git.juancwu.dev/juancwu/budgit/internal/testutil"
	"github.com/stretchr/testify/assert"
)

// testServices holds all services needed by tests, constructed once per DB.
type testServices struct {
	spaceSvc            *service.SpaceService
	tagSvc              *service.TagService
	listSvc             *service.ShoppingListService
	expenseSvc          *service.ExpenseService
	inviteSvc           *service.InviteService
	accountSvc          *service.MoneyAccountService
	methodSvc           *service.PaymentMethodService
	recurringSvc        *service.RecurringExpenseService
	budgetSvc           *service.BudgetService
	reportSvc           *service.ReportService
	loanSvc             *service.LoanService
	receiptSvc          *service.ReceiptService
	recurringReceiptSvc *service.RecurringReceiptService
}

func newTestServices(t *testing.T, dbi testutil.DBInfo) *testServices {
	t.Helper()
	spaceRepo := repository.NewSpaceRepository(dbi.DB)
	tagRepo := repository.NewTagRepository(dbi.DB)
	listRepo := repository.NewShoppingListRepository(dbi.DB)
	itemRepo := repository.NewListItemRepository(dbi.DB)
	expenseRepo := repository.NewExpenseRepository(dbi.DB)
	profileRepo := repository.NewProfileRepository(dbi.DB)
	inviteRepo := repository.NewInvitationRepository(dbi.DB)
	accountRepo := repository.NewMoneyAccountRepository(dbi.DB)
	methodRepo := repository.NewPaymentMethodRepository(dbi.DB)
	recurringRepo := repository.NewRecurringExpenseRepository(dbi.DB)
	budgetRepo := repository.NewBudgetRepository(dbi.DB)
	userRepo := repository.NewUserRepository(dbi.DB)
	loanRepo := repository.NewLoanRepository(dbi.DB)
	receiptRepo := repository.NewReceiptRepository(dbi.DB)
	recurringReceiptRepo := repository.NewRecurringReceiptRepository(dbi.DB)
	emailSvc := service.NewEmailService(nil, "test@example.com", "http://localhost:9999", "Budgit Test", false)
	spaceSvc := service.NewSpaceService(spaceRepo)
	expenseSvc := service.NewExpenseService(expenseRepo)
	loanSvc := service.NewLoanService(loanRepo, receiptRepo)
	receiptSvc := service.NewReceiptService(receiptRepo, loanRepo, accountRepo)
	recurringReceiptSvc := service.NewRecurringReceiptService(recurringReceiptRepo, receiptSvc, loanRepo, profileRepo, spaceRepo)

	return &testServices{
		spaceSvc:            spaceSvc,
		tagSvc:              service.NewTagService(tagRepo),
		listSvc:             service.NewShoppingListService(listRepo, itemRepo),
		expenseSvc:          expenseSvc,
		inviteSvc:           service.NewInviteService(inviteRepo, spaceRepo, userRepo, emailSvc),
		accountSvc:          service.NewMoneyAccountService(accountRepo),
		methodSvc:           service.NewPaymentMethodService(methodRepo),
		recurringSvc:        service.NewRecurringExpenseService(recurringRepo, expenseRepo, profileRepo, spaceRepo),
		budgetSvc:           service.NewBudgetService(budgetRepo),
		reportSvc:           service.NewReportService(expenseRepo),
		loanSvc:             loanSvc,
		receiptSvc:          receiptSvc,
		recurringReceiptSvc: recurringReceiptSvc,
	}
}

func TestListHandler_CreateList(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		svcs := newTestServices(t, dbi)
		h := NewListHandler(svcs.spaceSvc, svcs.listSvc)
		user, profile := testutil.CreateTestUserWithProfile(t, dbi.DB, "test@example.com", "Test")
		space := testutil.CreateTestSpace(t, dbi.DB, user.ID, "Test Space")

		req := testutil.NewAuthenticatedRequest(t, http.MethodPost, "/app/spaces/"+space.ID+"/lists", user, profile, url.Values{"name": {"Groceries"}})
		req.SetPathValue("spaceID", space.ID)

		w := httptest.NewRecorder()
		h.CreateList(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestListHandler_CreateList_EmptyName(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		svcs := newTestServices(t, dbi)
		h := NewListHandler(svcs.spaceSvc, svcs.listSvc)
		user, profile := testutil.CreateTestUserWithProfile(t, dbi.DB, "test@example.com", "Test")
		space := testutil.CreateTestSpace(t, dbi.DB, user.ID, "Test Space")

		req := testutil.NewAuthenticatedRequest(t, http.MethodPost, "/app/spaces/"+space.ID+"/lists", user, profile, url.Values{"name": {""}})
		req.SetPathValue("spaceID", space.ID)

		w := httptest.NewRecorder()
		h.CreateList(w, req)

		assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	})
}

func TestListHandler_DeleteList(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		svcs := newTestServices(t, dbi)
		h := NewListHandler(svcs.spaceSvc, svcs.listSvc)
		user, profile := testutil.CreateTestUserWithProfile(t, dbi.DB, "test@example.com", "Test")
		space := testutil.CreateTestSpace(t, dbi.DB, user.ID, "Test Space")
		list := testutil.CreateTestShoppingList(t, dbi.DB, space.ID, "Groceries")

		req := testutil.NewAuthenticatedRequest(t, http.MethodPost, "/app/spaces/"+space.ID+"/lists/"+list.ID+"?from=card", user, profile, nil)
		req.SetPathValue("spaceID", space.ID)
		req.SetPathValue("listID", list.ID)

		w := httptest.NewRecorder()
		h.DeleteList(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestListHandler_AddItemToList(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		svcs := newTestServices(t, dbi)
		h := NewListHandler(svcs.spaceSvc, svcs.listSvc)
		user, profile := testutil.CreateTestUserWithProfile(t, dbi.DB, "test@example.com", "Test")
		space := testutil.CreateTestSpace(t, dbi.DB, user.ID, "Test Space")
		list := testutil.CreateTestShoppingList(t, dbi.DB, space.ID, "Groceries")

		req := testutil.NewAuthenticatedRequest(t, http.MethodPost, "/app/spaces/"+space.ID+"/lists/"+list.ID+"/items", user, profile, url.Values{"name": {"Milk"}})
		req.SetPathValue("spaceID", space.ID)
		req.SetPathValue("listID", list.ID)

		w := httptest.NewRecorder()
		h.AddItemToList(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestTagHandler_CreateTag(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		svcs := newTestServices(t, dbi)
		h := NewTagHandler(svcs.spaceSvc, svcs.tagSvc)
		user, profile := testutil.CreateTestUserWithProfile(t, dbi.DB, "test@example.com", "Test")
		space := testutil.CreateTestSpace(t, dbi.DB, user.ID, "Test Space")

		req := testutil.NewAuthenticatedRequest(t, http.MethodPost, "/app/spaces/"+space.ID+"/tags", user, profile, url.Values{"name": {"food"}})
		req.SetPathValue("spaceID", space.ID)

		w := httptest.NewRecorder()
		h.CreateTag(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestTagHandler_DeleteTag(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		svcs := newTestServices(t, dbi)
		h := NewTagHandler(svcs.spaceSvc, svcs.tagSvc)
		user, profile := testutil.CreateTestUserWithProfile(t, dbi.DB, "test@example.com", "Test")
		space := testutil.CreateTestSpace(t, dbi.DB, user.ID, "Test Space")
		tag := testutil.CreateTestTag(t, dbi.DB, space.ID, "food", nil)

		req := testutil.NewAuthenticatedRequest(t, http.MethodPost, "/app/spaces/"+space.ID+"/tags/"+tag.ID, user, profile, nil)
		req.SetPathValue("spaceID", space.ID)
		req.SetPathValue("tagID", tag.ID)

		w := httptest.NewRecorder()
		h.DeleteTag(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestAccountHandler_CreateAccount(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		svcs := newTestServices(t, dbi)
		h := NewAccountHandler(svcs.spaceSvc, svcs.accountSvc, svcs.expenseSvc)
		user, profile := testutil.CreateTestUserWithProfile(t, dbi.DB, "test@example.com", "Test")
		space := testutil.CreateTestSpace(t, dbi.DB, user.ID, "Test Space")

		req := testutil.NewAuthenticatedRequest(t, http.MethodPost, "/app/spaces/"+space.ID+"/accounts", user, profile, url.Values{"name": {"Savings"}})
		req.SetPathValue("spaceID", space.ID)

		w := httptest.NewRecorder()
		h.CreateAccount(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestMethodHandler_CreatePaymentMethod(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		svcs := newTestServices(t, dbi)
		h := NewMethodHandler(svcs.spaceSvc, svcs.methodSvc)
		user, profile := testutil.CreateTestUserWithProfile(t, dbi.DB, "test@example.com", "Test")
		space := testutil.CreateTestSpace(t, dbi.DB, user.ID, "Test Space")

		req := testutil.NewAuthenticatedRequest(t, http.MethodPost, "/app/spaces/"+space.ID+"/payment-methods", user, profile, url.Values{
			"name":      {"Visa"},
			"type":      {"credit"},
			"last_four": {"4242"},
		})
		req.SetPathValue("spaceID", space.ID)

		w := httptest.NewRecorder()
		h.CreatePaymentMethod(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}
