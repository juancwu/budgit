package service

import (
	"errors"
	"time"

	"git.juancwu.dev/juancwu/budgit/internal/model"
	"git.juancwu.dev/juancwu/budgit/internal/repository"
)

var ErrInvalidTimezone = errors.New("invalid timezone")

type ProfileService struct {
	profileRepository repository.ProfileRepository
}

func NewProfileService(profileRepository repository.ProfileRepository) *ProfileService {
	return &ProfileService{
		profileRepository: profileRepository,
	}
}

func (s *ProfileService) ByUserID(userID string) (*model.Profile, error) {
	return s.profileRepository.ByUserID(userID)
}

func (s *ProfileService) UpdateTimezone(userID, timezone string) error {
	if _, err := time.LoadLocation(timezone); err != nil {
		return ErrInvalidTimezone
	}
	return s.profileRepository.UpdateTimezone(userID, timezone)
}
