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
	profileRepository    repository.ProfileRepository
	tokenRepository      repository.TokenRepository
	spaceService         *SpaceService
	jwtSecret            string
	jwtExpiry            time.Duration
	tokenMagicLinkExpiry time.Duration
	isProduction         bool
}

func NewAuthService(
	emailService *EmailService,
	userRepository repository.UserRepository,
	profileRepository repository.ProfileRepository,
	tokenRepository repository.TokenRepository,
	spaceService *SpaceService,
	jwtSecret string,
	jwtExpiry time.Duration,
	tokenMagicLinkExpiry time.Duration,
	isProduction bool,
) *AuthService {
	return &AuthService{
		emailService:         emailService,
		userRepository:       userRepository,
		profileRepository:    profileRepository,
		tokenRepository:      tokenRepository,
		spaceService:         spaceService,
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

func (s *AuthService) SetJWTCookie(w http.ResponseWriter, token string, expiry time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name:     "auth_token",
		Value:    token,
		Expires:  expiry,
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
		// User doesn't exists - create a new passwordless account
		if errors.Is(err, repository.ErrUserNotFound) {
			now := time.Now()
			user = &model.User{
				ID:        uuid.NewString(),
				Email:     email,
				CreatedAt: now,
			}
			_, err := s.userRepository.Create(user)
			if err != nil {
				return fmt.Errorf("failed to create user: %w", err)
			}

			slog.Info("new user created with id", "id", user.ID)

			profile := &model.Profile{
				ID:        uuid.NewString(),
				UserID:    user.ID,
				Name:      "",
				CreatedAt: now,
				UpdatedAt: now,
			}

			_, err = s.profileRepository.Create(profile)
			if err != nil {
				return fmt.Errorf("failed to create profile: %w", err)
			}

			_, err = s.spaceService.EnsurePersonalSpace(user)
			if err != nil {
				// Log the error but don't fail the whole auth flow
				slog.Error("failed to create personal space for new user", "error", err, "user_id", user.ID)
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

	profile, err := s.profileRepository.ByUserID(user.ID)
	name := ""
	if err == nil && profile != nil {
		name = profile.Name
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
	profile, err := s.profileRepository.ByUserID(userID)
	if err != nil {
		return false, fmt.Errorf("failed to get profile: %w", err)
	}

	return profile.Name == "", nil
}

// CompleteOnboarding sets the user's name during onboarding
func (s *AuthService) CompleteOnboarding(userID, name string) error {
	name = strings.TrimSpace(name)

	err := validation.ValidateName(name)
	if err != nil {
		return err
	}

	err = s.profileRepository.UpdateName(userID, name)
	if err != nil {
		return fmt.Errorf("failed to update profile: %w", err)
	}

	user, err := s.userRepository.ByID(userID)
	if err == nil {
		err = s.emailService.SendWelcomeEmail(user.Email, name)
		if err != nil {
			slog.Warn("failed to send welcome email", "error", err, "email", user.Email)
		}
	}

	slog.Info("onboarding completed", "user_id", user.ID, "name", name)

	return nil
}
