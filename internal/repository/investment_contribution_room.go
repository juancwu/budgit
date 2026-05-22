package repository

import (
	"database/sql"
	"errors"

	"git.juancwu.dev/juancwu/budgit/internal/model"
	"github.com/jmoiron/sqlx"
)

var ErrContributionRoomNotFound = errors.New("contribution room not found")

type InvestmentContributionRoomRepository interface {
	Upsert(room *model.InvestmentContributionRoom) error
	ByAccountAndYear(accountID string, year int) (*model.InvestmentContributionRoom, error)
	ByAccountID(accountID string) ([]*model.InvestmentContributionRoom, error)
	Delete(accountID string, year int) error
}

type investmentContributionRoomRepository struct {
	db *sqlx.DB
}

func NewInvestmentContributionRoomRepository(db *sqlx.DB) InvestmentContributionRoomRepository {
	return &investmentContributionRoomRepository{db: db}
}

func (r *investmentContributionRoomRepository) Upsert(room *model.InvestmentContributionRoom) error {
	query := `INSERT INTO investment_contribution_rooms (account_id, year, room_amount, created_at, updated_at)
	          VALUES ($1, $2, $3, $4, $5)
	          ON CONFLICT (account_id, year) DO UPDATE
	          SET room_amount = EXCLUDED.room_amount,
	              updated_at = EXCLUDED.updated_at;`
	_, err := r.db.Exec(query, room.AccountID, room.Year, room.RoomAmount, room.CreatedAt, room.UpdatedAt)
	return err
}

func (r *investmentContributionRoomRepository) ByAccountAndYear(accountID string, year int) (*model.InvestmentContributionRoom, error) {
	room := &model.InvestmentContributionRoom{}
	query := `SELECT * FROM investment_contribution_rooms WHERE account_id = $1 AND year = $2;`
	err := r.db.Get(room, query, accountID, year)
	if err == sql.ErrNoRows {
		return nil, ErrContributionRoomNotFound
	}
	if err != nil {
		return nil, err
	}
	return room, nil
}

func (r *investmentContributionRoomRepository) ByAccountID(accountID string) ([]*model.InvestmentContributionRoom, error) {
	var rooms []*model.InvestmentContributionRoom
	query := `SELECT * FROM investment_contribution_rooms WHERE account_id = $1 ORDER BY year DESC;`
	if err := r.db.Select(&rooms, query, accountID); err != nil {
		return nil, err
	}
	return rooms, nil
}

func (r *investmentContributionRoomRepository) Delete(accountID string, year int) error {
	res, err := r.db.Exec(`DELETE FROM investment_contribution_rooms WHERE account_id = $1 AND year = $2;`, accountID, year)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrContributionRoomNotFound
	}
	return nil
}

