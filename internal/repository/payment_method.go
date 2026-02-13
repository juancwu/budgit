package repository

import (
	"database/sql"
	"errors"
	"time"

	"git.juancwu.dev/juancwu/budgit/internal/model"
	"github.com/jmoiron/sqlx"
)

var (
	ErrPaymentMethodNotFound = errors.New("payment method not found")
)

type PaymentMethodRepository interface {
	Create(method *model.PaymentMethod) error
	GetByID(id string) (*model.PaymentMethod, error)
	GetBySpaceID(spaceID string) ([]*model.PaymentMethod, error)
	Update(method *model.PaymentMethod) error
	Delete(id string) error
}

type paymentMethodRepository struct {
	db *sqlx.DB
}

func NewPaymentMethodRepository(db *sqlx.DB) PaymentMethodRepository {
	return &paymentMethodRepository{db: db}
}

func (r *paymentMethodRepository) Create(method *model.PaymentMethod) error {
	query := `INSERT INTO payment_methods (id, space_id, name, type, last_four, created_by, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8);`
	_, err := r.db.Exec(query, method.ID, method.SpaceID, method.Name, method.Type, method.LastFour, method.CreatedBy, method.CreatedAt, method.UpdatedAt)
	return err
}

func (r *paymentMethodRepository) GetByID(id string) (*model.PaymentMethod, error) {
	method := &model.PaymentMethod{}
	query := `SELECT * FROM payment_methods WHERE id = $1;`
	err := r.db.Get(method, query, id)
	if err == sql.ErrNoRows {
		return nil, ErrPaymentMethodNotFound
	}
	return method, err
}

func (r *paymentMethodRepository) GetBySpaceID(spaceID string) ([]*model.PaymentMethod, error) {
	var methods []*model.PaymentMethod
	query := `SELECT * FROM payment_methods WHERE space_id = $1 ORDER BY created_at DESC;`
	err := r.db.Select(&methods, query, spaceID)
	if err != nil {
		return nil, err
	}
	return methods, nil
}

func (r *paymentMethodRepository) Update(method *model.PaymentMethod) error {
	method.UpdatedAt = time.Now()
	query := `UPDATE payment_methods SET name = $1, type = $2, last_four = $3, updated_at = $4 WHERE id = $5;`
	result, err := r.db.Exec(query, method.Name, method.Type, method.LastFour, method.UpdatedAt, method.ID)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err == nil && rows == 0 {
		return ErrPaymentMethodNotFound
	}
	return err
}

func (r *paymentMethodRepository) Delete(id string) error {
	query := `DELETE FROM payment_methods WHERE id = $1;`
	result, err := r.db.Exec(query, id)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err == nil && rows == 0 {
		return ErrPaymentMethodNotFound
	}
	return err
}
