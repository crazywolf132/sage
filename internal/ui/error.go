package ui

import "fmt"

// NewError creates a new error with the given message
func NewError(msg string) error {
	return fmt.Errorf(msg)
}
