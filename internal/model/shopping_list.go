package model

import "time"

type ShoppingList struct {
	ID        string    `db:"id"`
	SpaceID   string    `db:"space_id"`
	Name      string    `db:"name"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

type ListItem struct {
	ID        string    `db:"id"`
	ListID    string    `db:"list_id"`
	Name      string    `db:"name"`
	IsChecked bool      `db:"is_checked"`
	CreatedBy string    `db:"created_by"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}
