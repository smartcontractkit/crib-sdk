package domain

import (
	"errors"
)

var (
	// ErrSkipFile is an error that indicates that a file should be skipped.
	ErrSkipFile = errors.New("skip file")
	// ErrReadOnlyFileSystem is an error that indicates that the file system is read-only.
	ErrReadOnlyFileSystem = errors.New("read-only file system")
)

// SkipFileError is an error that indicates that a file should be skipped.
type SkipFileError struct {
	// Path is the path of the file that was skipped.
	Path string
}

func (e *SkipFileError) Error() string {
	return "file skipped: " + e.Path
}

func (e *SkipFileError) Unwrap() error {
	return nil
}

func (e *SkipFileError) Is(target error) bool {
	return errors.Is(target, ErrSkipFile)
}

// NewSkipFileError creates a new SkipFileError with the given path.
func NewSkipFileError(path string) error {
	return &SkipFileError{Path: path}
}
