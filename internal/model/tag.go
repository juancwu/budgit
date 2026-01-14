package model

import "time"

type Tag struct {
	ID        string    `db:"id"`
	SpaceID   string    `db:"space_id"`
	Name      string    `db:"name"`
	Color     *string   `db:"color"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}
