package domain

import "errors"

// ErrNotFoundInPath is an error that indicates that a file was not found in the specified path.
var ErrNotFoundInPath = errors.New("not found in path")

// NotFoundInPathError is an error that indicates that a file was not found in the specified path.
type NotFoundInPathError struct {
	// Path is the path where the file was not found (usually just the name).
	Path string
}

func (e *NotFoundInPathError) Error() string {
	return "file not found in path: " + e.Path
}

func (e *NotFoundInPathError) Unwrap() error {
	return nil
}

func (e *NotFoundInPathError) Is(target error) bool {
	return errors.Is(target, ErrNotFoundInPath)
}

// NewNotFoundInPathError creates a new NotFoundInPathError with the given path.
func NewNotFoundInPathError(path string) error {
	return &NotFoundInPathError{Path: path}
}
