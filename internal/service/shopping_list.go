package service

import (
	"fmt"
	"strings"
	"time"

	"git.juancwu.dev/juancwu/budgit/internal/model"
	"git.juancwu.dev/juancwu/budgit/internal/repository"
	"github.com/google/uuid"
)

type ShoppingListService struct {
	listRepo repository.ShoppingListRepository
	itemRepo repository.ListItemRepository
}

func NewShoppingListService(listRepo repository.ShoppingListRepository, itemRepo repository.ListItemRepository) *ShoppingListService {
	return &ShoppingListService{
		listRepo: listRepo,
		itemRepo: itemRepo,
	}
}

// List methods
func (s *ShoppingListService) CreateList(spaceID, name string) (*model.ShoppingList, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("list name cannot be empty")
	}

	now := time.Now()
	list := &model.ShoppingList{
		ID:        uuid.NewString(),
		SpaceID:   spaceID,
		Name:      name,
		CreatedAt: now,
		UpdatedAt: now,
	}

	err := s.listRepo.Create(list)
	if err != nil {
		return nil, err
	}

	return list, nil
}

func (s *ShoppingListService) GetListsForSpace(spaceID string) ([]*model.ShoppingList, error) {
	return s.listRepo.GetBySpaceID(spaceID)
}

func (s *ShoppingListService) GetList(listID string) (*model.ShoppingList, error) {
	return s.listRepo.GetByID(listID)
}

func (s *ShoppingListService) UpdateList(listID, name string) (*model.ShoppingList, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("list name cannot be empty")
	}

	list, err := s.listRepo.GetByID(listID)
	if err != nil {
		return nil, err
	}

	list.Name = name

	err = s.listRepo.Update(list)
	if err != nil {
		return nil, err
	}

	return list, nil
}

func (s *ShoppingListService) DeleteList(listID string) error {
	// First delete all items in the list
	err := s.itemRepo.DeleteByListID(listID)
	if err != nil {
		return fmt.Errorf("failed to delete items in list: %w", err)
	}
	// Then delete the list itself
	return s.listRepo.Delete(listID)
}

// Item methods
func (s *ShoppingListService) AddItemToList(listID, name, createdBy string) (*model.ListItem, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("item name cannot be empty")
	}

	now := time.Now()
	item := &model.ListItem{
		ID:        uuid.NewString(),
		ListID:    listID,
		Name:      name,
		IsChecked: false,
		CreatedBy: createdBy,
		CreatedAt: now,
		UpdatedAt: now,
	}

	err := s.itemRepo.Create(item)
	if err != nil {
		return nil, err
	}

	return item, nil
}

func (s *ShoppingListService) GetItem(itemID string) (*model.ListItem, error) {
	return s.itemRepo.GetByID(itemID)
}

func (s *ShoppingListService) GetItemsForList(listID string) ([]*model.ListItem, error) {
	return s.itemRepo.GetByListID(listID)
}

const ItemsPerCardPage = 5

func (s *ShoppingListService) GetItemsForListPaginated(listID string, page int) ([]*model.ListItem, int, error) {
	total, err := s.itemRepo.CountByListID(listID)
	if err != nil {
		return nil, 0, err
	}

	totalPages := (total + ItemsPerCardPage - 1) / ItemsPerCardPage
	if totalPages < 1 {
		totalPages = 1
	}
	if page < 1 {
		page = 1
	}
	if page > totalPages {
		page = totalPages
	}

	offset := (page - 1) * ItemsPerCardPage
	items, err := s.itemRepo.GetByListIDPaginated(listID, ItemsPerCardPage, offset)
	if err != nil {
		return nil, 0, err
	}

	return items, totalPages, nil
}

func (s *ShoppingListService) UpdateItem(itemID, name string, isChecked bool) (*model.ListItem, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("item name cannot be empty")
	}

	item, err := s.itemRepo.GetByID(itemID)
	if err != nil {
		return nil, err
	}

	item.Name = name
	item.IsChecked = isChecked

	err = s.itemRepo.Update(item)
	if err != nil {
		return nil, err
	}

	return item, nil
}

func (s *ShoppingListService) CheckItem(itemID string) error {
	item, err := s.itemRepo.GetByID(itemID)
	if err != nil {
		return err
	}

	item.IsChecked = true

	return s.itemRepo.Update(item)
}

func (s *ShoppingListService) GetListsWithUncheckedItems(spaceID string) ([]model.ListWithUncheckedItems, error) {
	lists, err := s.listRepo.GetBySpaceID(spaceID)
	if err != nil {
		return nil, err
	}

	var result []model.ListWithUncheckedItems
	for _, list := range lists {
		items, err := s.itemRepo.GetByListID(list.ID)
		if err != nil {
			return nil, err
		}

		var unchecked []*model.ListItem
		for _, item := range items {
			if !item.IsChecked {
				unchecked = append(unchecked, item)
			}
		}

		if len(unchecked) > 0 {
			result = append(result, model.ListWithUncheckedItems{
				List:  list,
				Items: unchecked,
			})
		}
	}

	return result, nil
}

func (s *ShoppingListService) DeleteItem(itemID string) error {
	return s.itemRepo.Delete(itemID)
}
