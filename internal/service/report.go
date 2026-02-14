package service

import (
	"time"

	"git.juancwu.dev/juancwu/budgit/internal/model"
	"git.juancwu.dev/juancwu/budgit/internal/repository"
)

type ReportService struct {
	expenseRepo repository.ExpenseRepository
}

func NewReportService(expenseRepo repository.ExpenseRepository) *ReportService {
	return &ReportService{expenseRepo: expenseRepo}
}

type DateRange struct {
	Label string
	Key   string
	From  time.Time
	To    time.Time
}

func (s *ReportService) GetSpendingReport(spaceID string, from, to time.Time) (*model.SpendingReport, error) {
	byTag, err := s.expenseRepo.GetExpensesByTag(spaceID, from, to)
	if err != nil {
		return nil, err
	}

	daily, err := s.expenseRepo.GetDailySpending(spaceID, from, to)
	if err != nil {
		return nil, err
	}

	monthly, err := s.expenseRepo.GetMonthlySpending(spaceID, from, to)
	if err != nil {
		return nil, err
	}

	topExpenses, err := s.expenseRepo.GetTopExpenses(spaceID, from, to, 10)
	if err != nil {
		return nil, err
	}

	// Get tags and payment methods for top expenses
	ids := make([]string, len(topExpenses))
	for i, e := range topExpenses {
		ids[i] = e.ID
	}

	tagsMap, _ := s.expenseRepo.GetTagsByExpenseIDs(ids)
	methodsMap, _ := s.expenseRepo.GetPaymentMethodsByExpenseIDs(ids)

	topWithTags := make([]*model.ExpenseWithTagsAndMethod, len(topExpenses))
	for i, e := range topExpenses {
		topWithTags[i] = &model.ExpenseWithTagsAndMethod{
			Expense:       *e,
			Tags:          tagsMap[e.ID],
			PaymentMethod: methodsMap[e.ID],
		}
	}

	totalIncome, totalExpenses, err := s.expenseRepo.GetIncomeVsExpenseSummary(spaceID, from, to)
	if err != nil {
		return nil, err
	}

	return &model.SpendingReport{
		ByTag:           byTag,
		DailySpending:   daily,
		MonthlySpending: monthly,
		TopExpenses:     topWithTags,
		TotalIncome:     totalIncome,
		TotalExpenses:   totalExpenses,
		NetBalance:      totalIncome - totalExpenses,
	}, nil
}

func GetPresetDateRanges(now time.Time) []DateRange {
	thisMonthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	thisMonthEnd := thisMonthStart.AddDate(0, 1, -1)
	thisMonthEnd = time.Date(thisMonthEnd.Year(), thisMonthEnd.Month(), thisMonthEnd.Day(), 23, 59, 59, 0, now.Location())

	lastMonthStart := thisMonthStart.AddDate(0, -1, 0)
	lastMonthEnd := thisMonthStart.AddDate(0, 0, -1)
	lastMonthEnd = time.Date(lastMonthEnd.Year(), lastMonthEnd.Month(), lastMonthEnd.Day(), 23, 59, 59, 0, now.Location())

	last3MonthsStart := thisMonthStart.AddDate(0, -2, 0)

	yearStart := time.Date(now.Year(), 1, 1, 0, 0, 0, 0, now.Location())

	return []DateRange{
		{Label: "This Month", Key: "this_month", From: thisMonthStart, To: thisMonthEnd},
		{Label: "Last Month", Key: "last_month", From: lastMonthStart, To: lastMonthEnd},
		{Label: "Last 3 Months", Key: "last_3_months", From: last3MonthsStart, To: thisMonthEnd},
		{Label: "This Year", Key: "this_year", From: yearStart, To: thisMonthEnd},
	}
}
