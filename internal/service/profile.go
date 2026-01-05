package service

import (
	"git.juancwu.dev/juancwu/budgit/internal/model"
	"git.juancwu.dev/juancwu/budgit/internal/repository"
)

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
