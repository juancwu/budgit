package model

import "time"

type Profile struct {
	ID        string    `db:"id"`
	UserID    string    `db:"user_id"`
	Name      string    `db:"name"`
	Timezone  *string   `db:"timezone"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

// Location returns the *time.Location for this profile's timezone.
// Returns UTC if timezone is nil or invalid.
func (p *Profile) Location() *time.Location {
	if p.Timezone == nil || *p.Timezone == "" {
		return time.UTC
	}
	loc, err := time.LoadLocation(*p.Timezone)
	if err != nil {
		return time.UTC
	}
	return loc
}
