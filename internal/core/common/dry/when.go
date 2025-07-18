package dry

// When acts as simple-ish ternary operator.
//
// Note: It is not technically a ternary operator as the truthy value is evaluated first.
func When[T any](cond bool, t, f T) T {
	if cond {
		return t
	}
	return f
}
