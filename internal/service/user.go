package service

import (
	"git.juancwu.dev/juancwu/budgit/internal/model"
	"git.juancwu.dev/juancwu/budgit/internal/repository"
)

type UserService struct {
	userRepository repository.UserRepository
}

func NewUserService(userRepository repository.UserRepository) *UserService {
	return &UserService{
		userRepository: userRepository,
	}
}

func (s *UserService) ByID(id string) (*model.User, error) {
	user, err := s.userRepository.ByID(id)
	if err != nil {
		return nil, err
	}

	return user, nil
}
