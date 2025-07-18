package crib

import (
	"github.com/smartcontractkit/crib-sdk/internal"
)

// SetValueAtPath sets a value at the specified path in a nested map structure.
// The path is a dot-separated string representing the nested keys.
// For example: "a.b.c" will set the value at map["a"]["b"]["c"].
// This function also supports array indexing like "containers[0].env[0].value".
func SetValueAtPath(values map[string]any, path string, value any) map[string]any {
	return internal.SetValueAtPath(values, path, value)
}
