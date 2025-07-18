package dry

import (
	"errors"
	"fmt"
)

// cError is a simple error wrapper to satisfy both the error and unwrap interfaces.
type cError struct {
	err error
}

func (e *cError) Error() string {
	return e.err.Error()
}

func (e *cError) Unwrap() error {
	return e.err
}

func (e *cError) Is(err error) bool {
	return errors.Is(e.err, err)
}

// ErrorAs is a helper utility that simplifies the process of checking if an error is of a specific type.
// Using the standard library:
//
//	var me *MyError
//	if errors.As(err, &me) {
//		fmt.Println(me.A, err)
//	}
//
// Using this utility:
//
//	if err, ok := dry.ErrorAs[*MyError](err); ok {
//		fmt.Println(err.A, err)
//	}
func ErrorAs[T any](err error) (T, bool) {
	var zero T
	if err == nil {
		return zero, false
	}

	var target T
	if errors.As(err, &target) {
		return target, true
	}
	return zero, false
}

// FirstError returns the first non-nil error, or nil.
// This method should only be used when it is safe to check many errors at once.
func FirstError(errs ...error) error {
	for _, err := range errs {
		if err != nil {
			return Wrap(err)
		}
	}
	return nil
}

// FirstErrorFns returns the first non-nil error from a list of functions.
// It iterates through each function, executes it, and checks the returned error.
// The first non-nil error is returned, or nil if all functions return nil errors.
func FirstErrorFns(checks ...func() error) error {
	for _, check := range checks {
		if err := check(); err != nil {
			return Wrap(err)
		}
	}
	return nil
}

// FirstError2 has a similar behavior as FirstError, but it returns 2 variables.
// Each check within the list must return the same two types of variables.
// If the error is non-nil, the first variable is returned as a zero value.
func FirstError2[T any](checks ...func() (T, error)) (T, error) {
	var val T
	var err error

	for _, check := range checks {
		val, err = Wrap2[T](check())
		if err != nil {
			return val, err
		}
	}
	return val, nil
}

// Wrap wraps an error, returning nil if nil or the error if not nil.
// Do not pass a wrapped error to this method, use Wrapf instead.
//
//	// Bad
//	err := Wrap(fmt.Errorf("wrapped: %w", err)) // cError is not nil in this case.
//
//	// Good
//	err := Wrapf(err, "wrapped") // cError is nil in this case.
func Wrap(err error) error {
	if err == nil {
		return nil
	}
	return &cError{err: err}
}

// Wrapf wraps an error, returning nil if nil or the formatted error message + error if not nil.
// Do not pass a wrapped error created via fmt.Errorf to this method, it will prevent the nil check from happening
// due it not implementing the Unwrap interface. You can, however, pass a wrapped error created by Wrap/Wrapf:
//
//	// Bad
//	err := Wrapf(fmt.Errorf("wrapped: %w", err), "oh no") // cError is not nil in this case.
//
//	// Okay (double wrapped)
//	err := Wrapf(Wrap(err), "oh no") // err can be nil in this case.
func Wrapf(err error, format string, args ...any) error {
	if err == nil {
		return nil
	}
	msg := fmt.Sprintf(format, args...)
	return Wrap(fmt.Errorf("%s: %w", msg, err))
}

// Wrap2 acts like Wrap, but capable of returning 2 variables. When the error is not nil, a zero value
// of the same type is returned.
func Wrap2[A any](a A, err error) (A, error) {
	var zero A
	if err := Wrap(err); err != nil {
		return zero, err
	}
	return a, nil
}

// Wrapf2 acts like Wrapf, but capable of returning 2 variables. When the error is not nil, a zero value
// of the same type is returned. The error message is formatted with the provided format and args.
func Wrapf2[A any](a A, err error, format string, args ...any) (A, error) {
	var zero A
	if err := Wrapf(err, format, args...); err != nil {
		return zero, err
	}
	return a, nil
}
