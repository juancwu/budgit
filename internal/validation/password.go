package validation

import "errors"

func ValidatePassword(password string) error {
	if password == "" {
		return errors.New("password is required")
	}

	if len(password) < 12 {
		return errors.New("password must be at least 12 characters")
	}

	return nil
}
