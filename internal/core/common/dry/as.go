package dry

// As will attempt to assert that the value is of the type T. If the assertion fails, it will return a zero value of
// the type T.
//
// Usages:
//
//	for k, v := range assure.As[map[string]any](t) { }
//	v := assure.As[int](1)
//	v := assure.As[*ComplexType](v)
func As[T any](t any) T {
	var v T
	if vv, ok := t.(T); ok {
		return vv
	}
	return v
}

// MustAs will attempt to assert that the value is of the type T. If the assertion fails, it will panic.
// This is useful for cases where you are sure that the value is of the type T, and you want to avoid
// the overhead of checking the type.
//
// Usages:
//
//	v := MustAs[int](1)
//	v := MustAs[*ComplexType](v)
func MustAs[T any](t any) T {
	if vv, ok := t.(T); ok {
		return vv
	}
	panic("type assertion failed")
}
