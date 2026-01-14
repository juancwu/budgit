package service

import (
	"fmt"
	"strings"
	"time"

	"git.juancwu.dev/juancwu/budgit/internal/model"
	"git.juancwu.dev/juancwu/budgit/internal/repository"
	"github.com/google/uuid"
)

type TagService struct {
	tagRepo repository.TagRepository
}

func NewTagService(tagRepo repository.TagRepository) *TagService {
	return &TagService{tagRepo: tagRepo}
}

func (s *TagService) CreateTag(spaceID, name string, color *string) (*model.Tag, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("tag name cannot be empty")
	}

	now := time.Now()
	tag := &model.Tag{
		ID:        uuid.NewString(),
		SpaceID:   spaceID,
		Name:      name,
		Color:     color,
		CreatedAt: now,
		UpdatedAt: now,
	}

	err := s.tagRepo.Create(tag)
	if err != nil {
		return nil, err
	}

	return tag, nil
}

func (s *TagService) GetTagsForSpace(spaceID string) ([]*model.Tag, error) {
	return s.tagRepo.GetBySpaceID(spaceID)
}

func (s *TagService) GetTagByID(id string) (*model.Tag, error) {
	return s.tagRepo.GetByID(id)
}

func (s *TagService) UpdateTag(id, name string, color *string) (*model.Tag, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("tag name cannot be empty")
	}

	tag, err := s.tagRepo.GetByID(id)
	if err != nil {
		return nil, err
	}

	tag.Name = name
	tag.Color = color

	err = s.tagRepo.Update(tag)
	if err != nil {
		return nil, err
	}

	return tag, nil
}

func (s *TagService) DeleteTag(id string) error {
	return s.tagRepo.Delete(id)
}
