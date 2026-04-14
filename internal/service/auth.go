package service

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"git.juancwu.dev/juancwu/budgit/internal/exception"
	"git.juancwu.dev/juancwu/budgit/internal/model"
	"git.juancwu.dev/juancwu/budgit/internal/repository"
	"git.juancwu.dev/juancwu/budgit/internal/validation"
	"github.com/alexedwards/argon2id"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
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
	emailService         *EmailService
	userRepository       repository.UserRepository
	tokenRepository      repository.TokenRepository
	spaceService         *SpaceService
	accountService       *AccountService
	jwtSecret            string
	jwtExpiry            time.Duration
	tokenMagicLinkExpiry time.Duration
	isProduction         bool
}

func NewAuthService(
	emailService *EmailService,
	userRepository repository.UserRepository,
	tokenRepository repository.TokenRepository,
	spaceService *SpaceService,
	accountService *AccountService,
	jwtSecret string,
	jwtExpiry time.Duration,
	tokenMagicLinkExpiry time.Duration,
	isProduction bool,
) *AuthService {
	return &AuthService{
		emailService:         emailService,
		userRepository:       userRepository,
		tokenRepository:      tokenRepository,
		spaceService:         spaceService,
		accountService:       accountService,
		jwtSecret:            jwtSecret,
		jwtExpiry:            jwtExpiry,
		tokenMagicLinkExpiry: tokenMagicLinkExpiry,
		isProduction:         isProduction,
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

	err = s.ComparePassword(password, *user.PasswordHash)
	if err != nil {
		return nil, e.WithError(ErrInvalidCredentials)
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

func (s *AuthService) SetPassword(userID, currentPassword, newPassword, confirmPassword string) error {
	e := exception.New("AuthService.SetPassword")

	user, err := s.userRepository.ByID(userID)
	if err != nil {
		return e.WithError(err)
	}

	// If user already has a password, verify current password
	if user.HasPassword() {
		err = s.ComparePassword(currentPassword, *user.PasswordHash)
		if err != nil {
			return e.WithError(ErrInvalidCredentials)
		}
	}

	if newPassword != confirmPassword {
		return e.WithError(ErrPasswordsDoNotMatch)
	}

	err = validation.ValidatePassword(newPassword)
	if err != nil {
		return e.WithError(ErrWeakPassword)
	}

	hashed, err := s.HashPassword(newPassword)
	if err != nil {
		return e.WithError(err)
	}

	user.PasswordHash = &hashed
	err = s.userRepository.Update(user)
	if err != nil {
		return e.WithError(err)
	}

	return nil
}

func (s *AuthService) GenerateJWT(user *model.User) (string, error) {
	claims := jwt.MapClaims{
		"user_id": user.ID,
		"email":   user.Email,
		"exp":     time.Now().Add(s.jwtExpiry).Unix(),
		"iat":     time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	tokenString, err := token.SignedString([]byte(s.jwtSecret))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

func (s *AuthService) VerifyJWT(tokenString string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(s.jwtSecret), nil
	})
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token")
}

func (s *AuthService) SetJWTCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     "auth_token",
		Value:    token,
		Expires:  time.Now().Add(s.jwtExpiry),
		Path:     "/",
		HttpOnly: true,
		Secure:   s.isProduction,
		SameSite: http.SameSiteLaxMode,
	})
}

func (s *AuthService) ClearJWTCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     "auth_token",
		Value:    "",
		Expires:  time.Unix(0, 0),
		Path:     "/",
		HttpOnly: true,
		Secure:   s.isProduction,
		SameSite: http.SameSiteLaxMode,
	})
}

func (s *AuthService) GenerateToken() (string, error) {
	bytes := make([]byte, 32)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func (s *AuthService) SendMagicLink(email string) error {
	email = strings.TrimSpace(strings.ToLower(email))

	err := validation.ValidateEmail(email)
	if err != nil {
		return ErrInvalidEmail
	}

	user, err := s.userRepository.ByEmail(email)
	if err != nil {
		// User doesn't exist - create a new passwordless account
		if errors.Is(err, repository.ErrUserNotFound) {
			now := time.Now()
			user = &model.User{
				ID:        uuid.NewString(),
				Email:     email,
				CreatedAt: now,
				UpdatedAt: now,
			}
			_, err := s.userRepository.Create(user)
			if err != nil {
				return fmt.Errorf("failed to create user: %w", err)
			}

			slog.Info("new passwordless user created", "email", email, "user_id", user.ID)
		} else {
			// user look up unexpected error
			return fmt.Errorf("failed to look up user: %w", err)
		}
	}

	err = s.tokenRepository.DeleteByUserAndType(user.ID, model.TokenTypeMagicLink)
	if err != nil {
		slog.Warn("failed to delete old magic link tokens", "error", err, "user_id", user.ID)
	}

	magicToken, err := s.GenerateToken()
	if err != nil {
		return fmt.Errorf("failed to generate token: %w", err)
	}

	token := &model.Token{
		ID:        uuid.NewString(),
		UserID:    user.ID,
		Type:      model.TokenTypeMagicLink,
		Token:     magicToken,
		ExpiresAt: time.Now().Add(s.tokenMagicLinkExpiry),
	}

	_, err = s.tokenRepository.Create(token)
	if err != nil {
		return fmt.Errorf("failed to create token: %w", err)
	}

	name := ""
	if user.Name != nil {
		name = *user.Name
	}

	err = s.emailService.SendMagicLinkEmail(user.Email, magicToken, name)
	if err != nil {
		slog.Error("failed to send magic link email", "error", err, "email", user.Email)
		return fmt.Errorf("failed to send email: %w", err)
	}

	slog.Info("magic link sent", "email", user.Email)
	return nil
}

func (s *AuthService) VerifyMagicLink(tokenString string) (*model.User, error) {
	token, err := s.tokenRepository.ConsumeToken(tokenString)
	if err != nil {
		return nil, fmt.Errorf("invalid or expired magic link")
	}

	if token.Type != model.TokenTypeMagicLink {
		return nil, fmt.Errorf("invalid token type")
	}

	user, err := s.userRepository.ByID(token.UserID)
	if errors.Is(err, repository.ErrUserNotFound) {
		return nil, err
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	if user.EmailVerifiedAt == nil {
		now := time.Now()
		user.EmailVerifiedAt = &now
		err = s.userRepository.Update(user)
		if err != nil {
			slog.Warn("failed to set email verification time", "error", err, "user_id", user.ID)
		}
	}

	slog.Info("user authenticated via magic link", "user_id", user.ID, "email", user.Email)

	return user, nil
}

// NeedsOnboarding checks if user needs to complete onboarding (name not set)
func (s *AuthService) NeedsOnboarding(userID string) (bool, error) {
	user, err := s.userRepository.ByID(userID)
	if err != nil {
		return false, fmt.Errorf("failed to get user: %w", err)
	}

	return user.Name == nil || *user.Name == "", nil
}

// CompleteOnboarding finalizes a user's onboarding by provisioning their first
// space and its default account, then saving their display name.
//
// The user-name update happens LAST so that if any step fails partway through,
// NeedsOnboarding still returns true and the user is routed back to retry.
// A retry is idempotent: if the user already has a space, the provisioning
// steps are skipped and only the name update runs.
func (s *AuthService) CompleteOnboarding(userID, name string) error {
	name = strings.TrimSpace(name)
	if err := validation.ValidateName(name); err != nil {
		return err
	}

	user, err := s.userRepository.ByID(userID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	existing, err := s.spaceService.GetSpacesForUser(userID)
	if err != nil {
		return fmt.Errorf("failed to check existing spaces: %w", err)
	}

	if len(existing) == 0 {
		space, err := s.spaceService.CreateSpace(DefaultSpaceName, userID)
		if err != nil {
			return fmt.Errorf("failed to create onboarding space: %w", err)
		}

		if _, err := s.accountService.CreateAccount(space.ID, DefaultAccountName); err != nil {
			if delErr := s.spaceService.DeleteSpace(space.ID); delErr != nil {
				slog.Error("failed to roll back space after account creation error",
					"space_id", space.ID, "error", delErr)
			}
			return fmt.Errorf("failed to create default account: %w", err)
		}
	}

	user.Name = &name
	if err := s.userRepository.Update(user); err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	if err := s.emailService.SendWelcomeEmail(user.Email, name); err != nil {
		slog.Warn("failed to send welcome email", "error", err, "email", user.Email)
	}

	slog.Info("onboarding completed",
		"user_id", user.ID, "name", name, "provisioned_space", len(existing) == 0)

	return nil
}
