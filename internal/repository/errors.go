package repository

import "errors"

// Repository errors
var (
	// ErrProjectNotFound is returned when a project is not found
	ErrProjectNotFound = errors.New("project not found")

	// ErrTodoNotFound is returned when a todo is not found
	ErrTodoNotFound = errors.New("todo not found")

	// ErrInvalidStatus is returned when an invalid task status is provided
	ErrInvalidStatus = errors.New("invalid task status")

	// ErrInvalidPosition is returned when an invalid position is provided
	ErrInvalidPosition = errors.New("invalid position")

	// ErrUserNotFound is returned when a user is not found
	ErrUserNotFound = errors.New("user not found")

	// ErrUnauthorized is returned when a user is not authorized to perform an action
	ErrUnauthorized = errors.New("unauthorized")

	// ErrDuplicateEntry is returned when trying to create a duplicate entry
	ErrDuplicateEntry = errors.New("duplicate entry")
)
