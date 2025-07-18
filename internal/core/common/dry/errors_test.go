package dry

import (
	"fmt"
	"testing"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/stretchr/testify/assert"
)

func ExampleFirstError_noErrors() {
	// This shows an example of when there are no errors in the chain.
	err := FirstError(
		nil,
		nil,
		nil,
	)
	fmt.Println(err)

	// Output:
	// <nil>
}

func ExampleFirstError_singleError() {
	// This shows an example of when the first error is returned.
	err := FirstError(
		nil,
		nil,
		assert.AnError,
	)
	fmt.Println(err)

	// Output:
	// assert.AnError general error for testing
}

func ExampleFirstError_multipleErrors() {
	// This shows an example of when there are multiple errors in the chain.
	err := FirstError(
		assert.AnError,
		nil,
		gofakeit.Error(),
	)
	fmt.Println(err)

	// Output:
	// assert.AnError general error for testing
}

func ExampleWrap() {
	err1 := Wrap(assert.AnError)
	err2 := Wrap(nil)

	fmt.Println(err1)
	fmt.Println(err2)

	// Output:
	// assert.AnError general error for testing
	// <nil>
}

func ExampleWrapf() {
	err1 := Wrapf(assert.AnError, "oh no")
	err2 := Wrapf(nil, "oh no")

	fmt.Println(err1)
	fmt.Println(err2)

	// Output:
	// oh no: assert.AnError general error for testing
	// <nil>
}

func ExampleWrapf_doubleWrap() {
	// Do not use fmt.Errorf to return a wrapped error!
	// Most errors do not implement the Unwrap interface so it's not possible
	// to check if the underlying error is nil without resorting to string comparison.
	var err error // nil error
	err2 := Wrapf(fmt.Errorf("wrapped: %w", err), "oh no")

	// Because we cannot Unwrap the error, this will return a non-nil error.
	fmt.Println(err2, err2 == nil)

	// You can, however, use the Wrap function to doubly wrap an error.
	err3 := Wrapf(Wrapf(err, "wrapped"), "oh no")
	fmt.Println(err3, err3 == nil)

	// Output:
	// oh no: wrapped: %!w(<nil>) false
	// <nil> true
}

func ExampleWrap2() {
	a1, err1 := Wrap2(1, assert.AnError)
	a2, err2 := Wrap2(1, nil)

	fmt.Println(a1, err1)
	fmt.Println(a2, err2)

	// Output:
	// 0 assert.AnError general error for testing
	// 1 <nil>
}

func ExampleWrapf2() {
	a1, err1 := Wrapf2(1, assert.AnError, "oh no")
	a2, err2 := Wrapf2(1, nil, "oh no")

	fmt.Println(a1, err1)
	fmt.Println(a2, err2)

	// Show a complex (pointer) example:
	type complexType struct {
		A int
	}
	a3, err3 := Wrapf2(&complexType{A: 1}, assert.AnError, "oh no")
	// Print the type to show that it is a pointer of the correct type.
	fmt.Printf("%v, %T, %v\n", a3, a3, err3)

	// Output:
	// 0 oh no: assert.AnError general error for testing
	// 1 <nil>
	// <nil>, *dry.complexType, oh no: assert.AnError general error for testing
}

type CustomError struct {
	Message string
}

func (c *CustomError) Error() string {
	return fmt.Sprintf("CustomError: %s", c.Message)
}

func ExampleErrorAs() {
	var (
		err1 error = &CustomError{Message: "This is a custom error"}
		err2 error = assert.AnError // This is not a CustomError
	)

	if err, ok := ErrorAs[*CustomError](err1); ok {
		fmt.Printf("err1 asserts correctly: %v\n", err)
		fmt.Printf("The fields can be accessed directly: %s\n", err.Message)
	} else {
		fmt.Println("err1 is not a custom error")
	}

	if err, ok := ErrorAs[*CustomError](err2); ok {
		fmt.Printf("err2 asserts correctly: %v\n", err)
		fmt.Printf("The fields can be accessed directly: %s\n", err.Message)
	} else {
		fmt.Println("err2 is not a custom error")
	}

	// Output:
	// err1 asserts correctly: CustomError: This is a custom error
	// The fields can be accessed directly: This is a custom error
	// err2 is not a custom error
}

func TestErrorAs(t *testing.T) {
	t.Parallel()

	t.Run("Implements", func(t *testing.T) {
		t.Parallel()

		err := Wrap(assert.AnError)
		newErr, ok := ErrorAs[*cError](err)
		assert.True(t, ok, "Expected error to implement cError")
		assert.NotNil(t, newErr, "Expected newErr to be non-nil")
	})
	t.Run("Does not implement", func(t *testing.T) {
		t.Parallel()

		err := assert.AnError
		newErr, ok := ErrorAs[*cError](err)
		assert.False(t, ok, "Expected error to not implement cError")
		assert.Nil(t, newErr, "Expected newErr to be nil")
	})
}

func TestFirstError(t *testing.T) {
	tests := []struct {
		desc string
		errs []error
		want error
	}{
		{
			desc: "no errors",
			errs: []error{nil, nil, nil},
			want: nil,
		},
		{
			desc: "first error",
			errs: []error{nil, nil, assert.AnError},
			want: assert.AnError,
		},
		{
			desc: "multiple errors",
			errs: []error{assert.AnError, nil, gofakeit.Error()},
			want: assert.AnError,
		},
	}
	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			t.Parallel()
			got := FirstError(tc.errs...)
			assert.ErrorIs(t, got, tc.want)
		})
	}
}

func TestFirstErrorFns(t *testing.T) {
	var i int
	fn := func() error {
		i++
		if i == 2 {
			return assert.AnError
		}
		return nil
	}

	err := FirstErrorFns(fn, fn, fn)
	assert.ErrorIs(t, err, assert.AnError)
	assert.Equal(t, 2, i, "Function should have been called twice before returning an error")
}

func TestFirstError2(t *testing.T) {
	tests := []struct {
		desc    string
		errs    []func() (string, error)
		want    string
		wantErr error
	}{
		{
			desc: "no errors",
			errs: []func() (string, error){
				func() (string, error) { return "a", nil },
				func() (string, error) { return "b", nil },
				func() (string, error) { return "c", nil },
			},
			want:    "c",
			wantErr: nil,
		},
		{
			desc: "first error",
			errs: []func() (string, error){
				func() (string, error) { return "a", nil },
				func() (string, error) { return "b", assert.AnError },
				func() (string, error) { return "c", nil },
			},
			want:    "",
			wantErr: assert.AnError,
		},
		{
			desc: "multiple errors",
			errs: []func() (string, error){
				func() (string, error) { return "a", assert.AnError },
				func() (string, error) { return "b", gofakeit.Error() },
			},
			want:    "",
			wantErr: assert.AnError,
		},
	}
	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			t.Parallel()

			got, err := FirstError2(tc.errs...)
			assert.Equal(t, tc.want, got)
			assert.ErrorIs(t, err, tc.wantErr)
		})
	}
}
