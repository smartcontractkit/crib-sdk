package domain

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSkipFileError(t *testing.T) {
	t.Parallel()

	is := assert.New(t)
	err := NewSkipFileError("test.txt")
	is.NotNil(err)
	is.Contains(err.Error(), "test.txt")
	is.True(errors.Is(err, ErrSkipFile))

	var skipErr *SkipFileError
	is.True(errors.As(err, &skipErr))

	err = errors.Join(err, assert.AnError)
	is.NotNil(err)
	is.Contains(err.Error(), "test.txt")
	is.True(errors.Is(err, ErrSkipFile))
}
