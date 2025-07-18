package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewNotFoundInPathError(t *testing.T) {
	t.Parallel()

	t.Run("is error", func(t *testing.T) {
		t.Parallel()
		is := assert.New(t)

		err := NewNotFoundInPathError("test.txt")
		is.Error(err, "expected error to be returned")
		is.Contains(err.Error(), "test.txt", "expected error message to contain 'test.txt'")
		is.ErrorIs(err, ErrNotFoundInPath)
	})
	t.Run("is not error", func(t *testing.T) {
		t.Parallel()
		is := assert.New(t)

		err := assert.AnError
		is.NotNil(err, "expected error to be returned")
		is.NotContains(err.Error(), "test.txt", "expected error message to not contain 'test.txt'")
		is.NotErrorIs(err, ErrNotFoundInPath, "expected error to not be nil")
	})
	t.Run("as error", func(t *testing.T) {
		t.Parallel()
		is := assert.New(t)

		err := NewNotFoundInPathError("test.txt")
		var notFoundErr *NotFoundInPathError
		is.ErrorAs(err, &notFoundErr, "expected error to be of type NotFoundInPathError")
		is.Equal("test.txt", notFoundErr.Path, "expected error path to be 'test.txt'")
	})
	t.Run("as not found in path error", func(t *testing.T) {
		t.Parallel()
		is := assert.New(t)

		err := assert.AnError
		var notFoundErr *NotFoundInPathError
		is.NotErrorAs(err, &notFoundErr, "expected error to be of type NotFoundInPathError")
	})
}
