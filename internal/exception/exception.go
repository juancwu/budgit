package exception

import "fmt"

type Exception struct {
	Operation string
}

func New(operation string) *Exception {
	return &Exception{Operation: operation}
}

func (e *Exception) WithError(err error) error {
	return fmt.Errorf("%s: %w", e.Operation, err)
}
