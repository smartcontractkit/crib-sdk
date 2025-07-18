package domain

import (
	"errors"
	"fmt"
)

// Action represents the type of action to be performed by the client-side apply manifest.
const (
	ActionAws          = "aws"
	ActionCmd          = "cmd"
	ActionCribctl      = "cribctl"
	ActionDocker       = "docker"
	ActionHelm         = "helm"
	ActionKind         = "kind"
	ActionKubectl      = "kubectl"
	ActionTask         = "task"
	ActionTelepresence = "telepresence"
)

// FailureAction represents the actions to be taken on failure during client-side apply.
const (
	FailureContinue = "continue"
	FailureAbort    = "abort"
)

var (
	// ErrEmptyAction is an error that indicates that the action field in the ClientSideApplySpec is empty.
	ErrEmptyAction = errors.New("action cannot be empty")
	// ErrAbort is an error that indicates that the previous step failed and that the handler should abort any further processing.
	ErrAbort = errors.New("abort error: previous step failed, aborting further processing")
	// ErrContinue is an error that indicates that the previous step failed but that the handler should continue processing the next step.
	ErrContinue = errors.New("continue error: previous step failed, continuing to next step")
)

// ClientSideApplyManifest represents a manifest that is intended to be applied on the client side.
type (
	ClientSideApplyManifest struct {
		Manifest `yaml:",inline"`
		Spec     ClientSideApplySpec `yaml:"spec"`
	}

	ClientSideApplySpec struct {
		OnFailure string   `yaml:"onFailure" validate:"required,oneof=continue abort"`
		Action    string   `yaml:"action"    validate:"required,oneof=aws cmd cribctl docker helm kind kubectl task telepresence"`
		Args      []string `yaml:"args"      validate:"required,dive"`
	}

	// RunnerResult represents the result of a client-side apply operation.
	RunnerResult struct {
		Output []byte
	}

	// AbortError is an error that indicates that the previous step failed and that the handler should
	// abort any further processing.
	AbortError struct {
		err error
	}

	// ContinueError is an error that indicates that the previous step failed but that the handler should
	// continue processing the next step.
	ContinueError struct {
		err error
	}
)

// Error implements the error interface for AbortError.
func (e *AbortError) Error() string {
	return e.err.Error()
}

// Error implements the error interface for ContinueError.
func (e *ContinueError) Error() string {
	return e.err.Error()
}

// Is checks if the error is of type AbortError.
func (e *AbortError) Is(target error) bool {
	var abortError *AbortError
	return errors.Is(target, ErrAbort) || errors.As(target, &abortError)
}

// Is checks if the error is of type ContinueError.
func (e *ContinueError) Is(target error) bool {
	var continueError *ContinueError
	return errors.Is(target, ErrContinue) || errors.As(target, &continueError)
}

// NewAbortError creates a new AbortError with the given message.
func NewAbortError(err error) error {
	return &AbortError{err: err}
}

// NewContinueError creates a new ContinueError with the given message.
func NewContinueError(err error) error {
	return &ContinueError{err: err}
}

// NewError creates a new appropriate error for the given OnFailure action.
func (m *ClientSideApplyManifest) NewError(err error) error {
	if err == nil {
		return nil
	}
	switch m.Spec.OnFailure {
	case FailureContinue:
		return NewContinueError(err)
	case FailureAbort:
		return NewAbortError(err)
	}
	return fmt.Errorf("unknown onFailure action: %q", m.Spec.OnFailure)
}
