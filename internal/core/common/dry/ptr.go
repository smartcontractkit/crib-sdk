package dry

import (
	"github.com/samber/lo"
)

// ToPtr returns a pointer to the value passed in.
func ToPtr[T any](v T) *T {
	return &v
}

// ToFloat64Ptr converts any numeric value to a float64 pointer.
// This is useful for Kubernetes API types that require float64 pointers.
func ToFloat64Ptr[T ~int | ~int32 | ~int64 | ~float32 | ~float64](val T) *float64 {
	f := float64(val)
	return &f
}

// FromPtr returns the value that the pointer points to. If the value is nil, a
// zero value of type T is returned.
func FromPtr[T any](v *T) T {
	if v == nil {
		return Empty[T]()
	}
	return *v
}

// Empty returns a zero value of type T.
func Empty[T any]() T {
	var zero T
	return zero
}

// PtrMapping returns a pointer to a map of pointers to the provided values.
// ie. map[string]string becomes *map[string]*string.
//
//nolint:gocritic // This type matches what cdk8s expects for pointers to maps.
func PtrMapping[K comparable, V any](m map[K]V) *map[K]*V {
	if m == nil {
		return nil
	}
	return ToPtr(lo.MapEntries(m, func(key K, value V) (K, *V) {
		return key, ToPtr(value)
	}))
}

// PtrSlice returns a pointer to a slice of pointers to all of the provided values.
func PtrSlice[T any](v []T) *[]*T {
	if v == nil {
		return nil
	}
	slice := make([]*T, len(v))
	for i := 0; i < len(v); i++ {
		slice[i] = ToPtr(v[i])
	}
	return ToPtr(slice)
}

// PtrStrings returns a pointer to a slice of pointers to all of the provided strings.
// idea from jsii.Strings https://github.com/aws/jsii-runtime-go/blob/f277e617fbbb42c24e5486f151f1ba67d5fd9999/helpers.go#L64
func PtrStrings(v ...string) *[]*string {
	return PtrSlice(v)
}
