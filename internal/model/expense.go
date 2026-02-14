package model

import "time"

type ExpenseType string

const (
	ExpenseTypeExpense ExpenseType = "expense"
	ExpenseTypeTopup   ExpenseType = "topup"
)

type Expense struct {
	ID                 string      `db:"id"`
	SpaceID            string      `db:"space_id"`
	CreatedBy          string      `db:"created_by"`
	Description        string      `db:"description"`
	AmountCents        int         `db:"amount_cents"`
	Type               ExpenseType `db:"type"`
	Date               time.Time   `db:"date"`
	PaymentMethodID    *string     `db:"payment_method_id"`
	RecurringExpenseID *string     `db:"recurring_expense_id"`
	CreatedAt          time.Time   `db:"created_at"`
	UpdatedAt          time.Time   `db:"updated_at"`
}

type ExpenseWithTags struct {
	Expense
	Tags []*Tag
}

type ExpenseWithTagsAndMethod struct {
	Expense
	Tags          []*Tag
	PaymentMethod *PaymentMethod
}

type ExpenseTag struct {
	ExpenseID string `db:"expense_id"`
	TagID     string `db:"tag_id"`
}

type ExpenseItem struct {
	ExpenseID string `db:"expense_id"`
	ItemID    string `db:"item_id"`
}

type TagExpenseSummary struct {
	TagID       string `db:"tag_id"`
	TagName     string `db:"tag_name"`
	TagColor    *string `db:"tag_color"`
	TotalAmount int    `db:"total_amount"`
}
