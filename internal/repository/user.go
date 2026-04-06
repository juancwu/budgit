package repository

import (
	"database/sql"
	"errors"
	"strings"

	"git.juancwu.dev/juancwu/budgit/internal/model"
	"github.com/jmoiron/sqlx"
)

var (
	ErrUserNotFound   = errors.New("user not found")
	ErrDuplicateEmail = errors.New("email already exists")
)

type UserRepository interface {
	Create(user *model.User) (string, error)
	ByID(id string) (*model.User, error)
	ByEmail(email string) (*model.User, error)
	Update(user *model.User) error
	Delete(id string) error
}

type userRepository struct {
	db *sqlx.DB
}

func NewUserRepository(db *sqlx.DB) UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) Create(user *model.User) (string, error) {
	query := `INSERT INTO users (id, email, name, password_hash, email_verified_at, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, $7);`

	_, err := r.db.Exec(query, user.ID, user.Email, user.Name, user.PasswordHash, user.EmailVerifiedAt, user.CreatedAt, user.UpdatedAt)
	if err != nil {
		errStr := err.Error()
		if strings.Contains(errStr, "UNIQUE constraint failed") || strings.Contains(errStr, "duplicate key value") {
			return "", ErrDuplicateEmail
		}
		return "", err
	}

	return user.ID, nil
}

func (r *userRepository) ByID(id string) (*model.User, error) {
	user := &model.User{}
	query := `SELECT * FROM users WHERE id = $1;`

	err := r.db.Get(user, query, id)
	if err == sql.ErrNoRows {
		return nil, ErrUserNotFound
	}

	return user, err
}

func (r *userRepository) ByEmail(email string) (*model.User, error) {
	var user model.User
	query := `SELECT * FROM users WHERE email = $1;`

	err := r.db.Get(&user, query, email)
	if err == sql.ErrNoRows {
		return nil, ErrUserNotFound
	}

	return &user, err
}

func (r *userRepository) Update(user *model.User) error {
	query := `UPDATE users SET email = $1, name = $2, password_hash = $3, pending_email = $4, email_verified_at = $5, updated_at = $6 WHERE id = $7;`

	_, err := r.db.Exec(query, user.Email, user.Name, user.PasswordHash, user.PendingEmail, user.EmailVerifiedAt, user.UpdatedAt, user.ID)

	return err
}

func (r *userRepository) Delete(id string) error {
	query := `DELETE FROM users WHERE id = $1;`

	result, err := r.db.Exec(query, id)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return ErrUserNotFound
	}

	return nil
}
