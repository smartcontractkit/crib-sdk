package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestClientSideApplyErrors(t *testing.T) {
	t.Parallel()

	t.Run(FailureAbort, func(t *testing.T) {
		t.Parallel()
		is := assert.New(t)

		err := NewAbortError(assert.AnError)
		is.Error(err, "expected error to be returned")
		is.ErrorIs(err, ErrAbort)
		var abortErr *AbortError
		is.ErrorAs(err, &abortErr, "expected error to be of type AbortError")
	})
	t.Run(FailureContinue, func(t *testing.T) {
		t.Parallel()
		is := assert.New(t)

		err := NewContinueError(assert.AnError)
		is.Error(err, "expected error to be returned")
		is.ErrorIs(err, ErrContinue)
		var continueErr *ContinueError
		is.ErrorAs(err, &continueErr, "expected error to be of type ContinueError")
	})
}
