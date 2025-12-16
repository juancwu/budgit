package service

import (
	"errors"
	"strings"

	"git.juancwu.dev/juancwu/budgething/internal/exception"
	"git.juancwu.dev/juancwu/budgething/internal/model"
	"git.juancwu.dev/juancwu/budgething/internal/repository"
	"github.com/alexedwards/argon2id"
)

var (
	ErrInvalidCredentials  = errors.New("invalid email or password")
	ErrNoPassword          = errors.New("account uses passwordless login. Use magic link")
	ErrPasswordsDoNotMatch = errors.New("passwords do not match")
	ErrEmailAlreadyExists  = errors.New("email already exists")
	ErrWeakPassword        = errors.New("password must be at least 12 characters")
	ErrCommonPassword      = errors.New("password is too common, please choose a stronger one")
	ErrEmailNotVerified    = errors.New("email not verified")
	ErrInvalidEmail        = errors.New("invalid email address")
	ErrNameRequired        = errors.New("name is required")
)

type AuthService struct {
	userRepository repository.UserRepository
}

func NewAuthService(userRepository repository.UserRepository) *AuthService {
	return &AuthService{
		userRepository: userRepository,
	}
}

func (s *AuthService) LoginWithPassword(email, password string) (*model.User, error) {
	e := exception.New("AuthService.LoginWithPassword")

	email = strings.TrimSpace(strings.ToLower(email))

	user, err := s.userRepository.ByEmail(email)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return nil, e.WithError(ErrInvalidCredentials)
		}
		return nil, e.WithError(err)
	}

	if !user.HasPassword() {
		return nil, e.WithError(ErrNoPassword)
	}

	return user, nil
}

func (s *AuthService) HashPassword(password string) (string, error) {
	e := exception.New("AuthService.HashPassword")

	hashed, err := argon2id.CreateHash(password, argon2id.DefaultParams)
	if err != nil {
		return "", e.WithError(err)
	}
	return hashed, nil
}

func (s *AuthService) ComparePassword(password, hash string) error {
	e := exception.New("AuthService.ComparePassword")
	match, err := argon2id.ComparePasswordAndHash(password, hash)
	if err != nil {
		return e.WithError(err)
	}
	if !match {
		return e.WithError(ErrPasswordsDoNotMatch)
	}
	return nil
}
