package service

import (
	"testing"
	"time"

	"git.juancwu.dev/juancwu/budgit/internal/model"
	"git.juancwu.dev/juancwu/budgit/internal/testutil"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTransactionService_CategoryTimeSeries(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		f := newTxnFixture(t, dbi)
		accountID := f.account.ID

		rent := testutil.CreateTestCategory(t, dbi.DB, accountID, "Rent")
		food := testutil.CreateTestCategory(t, dbi.DB, accountID, "Food")

		jan := time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC)
		feb := time.Date(2026, 2, 15, 12, 0, 0, 0, time.UTC)

		// Fund the account so the transfer has available balance.
		_, err := f.svc.Deposit(DepositInput{AccountID: f.account.ID, Title: "Seed", Amount: decimal.NewFromInt(10000), OccurredAt: jan, ActorID: f.user.ID})
		require.NoError(t, err)

		pay := func(title string, amount int64, cat string, when time.Time) {
			_, err := f.svc.PayBill(PayBillInput{AccountID: f.account.ID, Title: title, Amount: decimal.NewFromInt(amount), OccurredAt: when, CategoryID: cat, ActorID: f.user.ID})
			require.NoError(t, err)
		}
		pay("Rent Jan", 1000, rent.ID, jan)
		pay("Food Jan", 200, food.ID, jan)
		pay("Rent Feb", 1000, rent.ID, feb)
		pay("Misc Feb", 50, "", feb) // uncategorized

		// A transfer's withdrawal half must NOT count as spending.
		dest := testutil.CreateTestAccount(t, dbi.DB, f.account.SpaceID, "Savings")
		_, err = f.svc.Transfer(TransferInput{SourceAccountID: f.account.ID, DestAccountID: dest.ID, Title: "Move", Amount: decimal.NewFromInt(300), OccurredAt: jan, ActorID: f.user.ID})
		require.NoError(t, err)

		from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
		to := time.Date(2026, 2, 28, 23, 59, 59, 0, time.UTC)

		byName := func(ts *model.CategoryTimeSeries) map[string]model.CategorySeriesData {
			m := map[string]model.CategorySeriesData{}
			for _, s := range ts.Series {
				m[s.CategoryName] = s
			}
			return m
		}

		// Include uncategorized.
		ts, err := f.svc.CategoryTimeSeries(CategorySeriesInput{
			AccountID: accountID, Type: model.TransactionTypeWithdrawal,
			From: from, To: to, Granularity: "month", IncludeUncategorized: true,
		})
		require.NoError(t, err)
		require.Len(t, ts.Buckets, 2, "Jan and Feb buckets")
		assert.True(t, decimal.NewFromInt(2250).Equal(ts.Total), "2000 rent + 200 food + 50 misc; transfer excluded")

		m := byName(ts)
		require.Contains(t, m, "Rent")
		require.Contains(t, m, "Food")
		require.Contains(t, m, "Uncategorized")
		// Series ordered largest-first: Rent(2000), Food(200), Uncategorized(50).
		assert.Equal(t, "Rent", ts.Series[0].CategoryName)
		assert.True(t, decimal.NewFromInt(2000).Equal(m["Rent"].Total))
		assert.True(t, decimal.NewFromInt(1000).Equal(m["Rent"].Values[0]), "Rent Jan")
		assert.True(t, decimal.NewFromInt(1000).Equal(m["Rent"].Values[1]), "Rent Feb")
		assert.True(t, decimal.NewFromInt(200).Equal(m["Food"].Values[0]), "Food Jan")
		assert.True(t, decimal.Zero.Equal(m["Food"].Values[1]), "Food Feb zero-filled")
		assert.True(t, decimal.NewFromInt(50).Equal(m["Uncategorized"].Values[1]), "Misc Feb")

		// Exclude uncategorized.
		ts2, err := f.svc.CategoryTimeSeries(CategorySeriesInput{
			AccountID: accountID, Type: model.TransactionTypeWithdrawal,
			From: from, To: to, Granularity: "month", IncludeUncategorized: false,
		})
		require.NoError(t, err)
		assert.NotContains(t, byName(ts2), "Uncategorized")
		assert.True(t, decimal.NewFromInt(2200).Equal(ts2.Total))

		// Invalid granularity is rejected.
		_, err = f.svc.CategoryTimeSeries(CategorySeriesInput{AccountID: accountID, Type: model.TransactionTypeWithdrawal, From: from, To: to, Granularity: "week", IncludeUncategorized: true})
		assert.Error(t, err)
	})
}
