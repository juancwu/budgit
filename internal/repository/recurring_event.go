package repository

import (
	"database/sql"
	"errors"
	"time"

	"git.juancwu.dev/juancwu/budgit/internal/model"
	"github.com/jmoiron/sqlx"
)

var ErrRecurringEventNotFound = errors.New("recurring event not found")

type RecurringEventRepository interface {
	Create(e *model.RecurringEvent) error
	ByID(id string) (*model.RecurringEvent, error)
	BySpaceID(spaceID string) ([]*model.RecurringEvent, error)
	ByAccountID(accountID string) ([]*model.RecurringEvent, error)
	DueBefore(now time.Time) ([]*model.RecurringEvent, error)
	Update(e *model.RecurringEvent) error
	UpdateCursor(id string, nextRunAt time.Time, lastRunAt time.Time) error
	SetPaused(id string, paused bool) error
	Delete(id string) error
}

type recurringEventRepository struct {
	db *sqlx.DB
}

func NewRecurringEventRepository(db *sqlx.DB) RecurringEventRepository {
	return &recurringEventRepository{db: db}
}

func (r *recurringEventRepository) Create(e *model.RecurringEvent) error {
	query := `INSERT INTO recurring_events (
        id, space_id, kind, source_account_id, title, amount, description,
        frequency, interval_count, day_of_week, day_of_month, month_of_year,
        fire_hour, fire_minute, timezone,
        next_run_at, last_run_at, paused, created_at, updated_at
    ) VALUES (
        $1, $2, $3, $4, $5, $6, $7,
        $8, $9, $10, $11, $12,
        $13, $14, $15,
        $16, $17, $18, $19, $20
    );`
	_, err := r.db.Exec(query,
		e.ID, e.SpaceID, e.Kind, e.SourceAccountID, e.Title, e.Amount, e.Description,
		e.Frequency, e.IntervalCount, e.DayOfWeek, e.DayOfMonth, e.MonthOfYear,
		e.FireHour, e.FireMinute, e.Timezone,
		e.NextRunAt, e.LastRunAt, e.Paused, e.CreatedAt, e.UpdatedAt,
	)
	return err
}

func (r *recurringEventRepository) ByID(id string) (*model.RecurringEvent, error) {
	out := &model.RecurringEvent{}
	err := r.db.Get(out, `SELECT * FROM recurring_events WHERE id = $1;`, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrRecurringEventNotFound
	}
	return out, err
}

func (r *recurringEventRepository) BySpaceID(spaceID string) ([]*model.RecurringEvent, error) {
	var out []*model.RecurringEvent
	err := r.db.Select(&out, `SELECT * FROM recurring_events WHERE space_id = $1 ORDER BY created_at DESC;`, spaceID)
	return out, err
}

func (r *recurringEventRepository) ByAccountID(accountID string) ([]*model.RecurringEvent, error) {
	var out []*model.RecurringEvent
	query := `SELECT * FROM recurring_events
	          WHERE source_account_id = $1
	          ORDER BY created_at DESC;`
	err := r.db.Select(&out, query, accountID)
	return out, err
}

func (r *recurringEventRepository) DueBefore(now time.Time) ([]*model.RecurringEvent, error) {
	var out []*model.RecurringEvent
	query := `SELECT * FROM recurring_events
	          WHERE paused = FALSE AND next_run_at <= $1
	          ORDER BY next_run_at ASC;`
	err := r.db.Select(&out, query, now)
	return out, err
}

func (r *recurringEventRepository) Update(e *model.RecurringEvent) error {
	query := `UPDATE recurring_events SET
        kind = $1, source_account_id = $2, title = $3, amount = $4, description = $5,
        frequency = $6, interval_count = $7, day_of_week = $8, day_of_month = $9, month_of_year = $10,
        fire_hour = $11, fire_minute = $12, timezone = $13,
        next_run_at = $14, paused = $15, updated_at = CURRENT_TIMESTAMP
        WHERE id = $16;`
	res, err := r.db.Exec(query,
		e.Kind, e.SourceAccountID, e.Title, e.Amount, e.Description,
		e.Frequency, e.IntervalCount, e.DayOfWeek, e.DayOfMonth, e.MonthOfYear,
		e.FireHour, e.FireMinute, e.Timezone,
		e.NextRunAt, e.Paused, e.ID,
	)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrRecurringEventNotFound
	}
	return nil
}

func (r *recurringEventRepository) UpdateCursor(id string, nextRunAt time.Time, lastRunAt time.Time) error {
	query := `UPDATE recurring_events
	          SET next_run_at = $1, last_run_at = $2, updated_at = CURRENT_TIMESTAMP
	          WHERE id = $3;`
	res, err := r.db.Exec(query, nextRunAt, lastRunAt, id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrRecurringEventNotFound
	}
	return nil
}

func (r *recurringEventRepository) SetPaused(id string, paused bool) error {
	res, err := r.db.Exec(
		`UPDATE recurring_events SET paused = $1, updated_at = CURRENT_TIMESTAMP WHERE id = $2;`,
		paused, id,
	)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrRecurringEventNotFound
	}
	return nil
}

func (r *recurringEventRepository) Delete(id string) error {
	res, err := r.db.Exec(`DELETE FROM recurring_events WHERE id = $1;`, id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrRecurringEventNotFound
	}
	return nil
}
