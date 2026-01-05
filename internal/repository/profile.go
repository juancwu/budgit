package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"git.juancwu.dev/juancwu/budgit/internal/model"
	"github.com/jmoiron/sqlx"
)

var (
	ErrProfileNotFound = errors.New("profile not found")
)

type ProfileRepository interface {
	Create(profile *model.Profile) (string, error)
	ByUserID(userID string) (*model.Profile, error)
	UpdateName(userID, name string) error
}

type profileRepository struct {
	db *sqlx.DB
}

func NewProfileRepository(db *sqlx.DB) *profileRepository {
	return &profileRepository{db: db}
}

func (r *profileRepository) Create(profile *model.Profile) (string, error) {
	if profile.CreatedAt.IsZero() {
		profile.CreatedAt = time.Now()
	}
	if profile.UpdatedAt.IsZero() {
		profile.UpdatedAt = time.Now()
	}

	_, err := r.db.Exec(`
		INSERT INTO profiles (id, user_id, name, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
		`, profile.ID, profile.UserID, profile.Name, profile.CreatedAt, profile.UpdatedAt)
	if err != nil {
		return "", err
	}

	return profile.ID, nil
}

func (r *profileRepository) ByUserID(userID string) (*model.Profile, error) {
	var profile model.Profile
	err := r.db.Get(&profile, `SELECT * FROM profiles WHERE user_id = $1`, userID)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrProfileNotFound
	}
	if err != nil {
		return nil, err
	}

	return &profile, nil
}

func (r *profileRepository) UpdateName(userID, name string) error {
	result, err := r.db.Exec(`
		UPDATE profiles
		SET name = $1, updated_at = $2
		WHERE user_id = $3
		`, name, time.Now(), userID)

	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("no profile found for user_id: %s", userID)
	}

	return nil
}
