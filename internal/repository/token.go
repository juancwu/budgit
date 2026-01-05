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
	ErrTokenNotFound = errors.New("token not found")
)

type TokenRepository interface {
	Create(token *model.Token) (string, error)
	DeleteByUserAndType(userID string, tokenType string) error
	ConsumeToken(token string) (*model.Token, error)
}

type tokenRepository struct {
	db *sqlx.DB
}

func NewTokenRepository(db *sqlx.DB) *tokenRepository {
	return &tokenRepository{db: db}
}

func (r *tokenRepository) Create(token *model.Token) (string, error) {
	if token.CreatedAt.IsZero() {
		token.CreatedAt = time.Now()
	}

	query := `
	INSERT INTO tokens (id, user_id, type, token, expires_at, created_at)
	VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err := r.db.Exec(query, token.ID, token.UserID, token.Type, token.Token, token.ExpiresAt, token.CreatedAt)
	if err != nil {
		return "", fmt.Errorf("failed to create token: %w", err)
	}

	return token.ID, nil
}

func (r *tokenRepository) DeleteByUserAndType(userID string, tokenType string) error {
	query := `DELETE FROM tokens WHERE user_id = $1 AND type = $2 AND used_at IS NULL`
	_, err := r.db.Exec(query, userID, tokenType)
	return err
}

func (r *tokenRepository) ConsumeToken(tokenString string) (*model.Token, error) {
	var token model.Token
	now := time.Now()

	query := `
	UPDATE tokens
	SET used_at = $1
	WHERE token = $2
	AND used_at IS NULL
	AND expires_at > $3
	RETURNING *
	`

	err := r.db.Get(&token, query, now, tokenString, now)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrTokenNotFound
	}
	if err != nil {
		return nil, err
	}

	return &token, nil
}
