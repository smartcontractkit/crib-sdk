package crib

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/crib-sdk/internal/core/common/dry"
)

type MockProps struct {
	String *string             `json:"string,omitzero"`
	Int    *int                `json:"int,omitzero"`
	Bool   *bool               `json:"bool,omitzero"`
	Map    *map[string]*string `json:"map,omitzero"`
}

func (m *MockProps) Validate(context.Context) error {
	return nil
}

func TestResourceID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		prefix   string
		props    Props
		expected string
	}{
		{
			name:     "nil props",
			prefix:   "foo",
			expected: "foo-5b9bc4ba",
		},
		{
			name:   "empty prefix",
			prefix: "",
			props: &MockProps{
				String: dry.ToPtr("foo-bar-baz"),
			},
			expected: "unknown-19922604",
		},
		{
			name:     "empty prefix with empty props",
			prefix:   "",
			props:    &MockProps{},
			expected: "unknown-08f44b07",
		},
		{
			name:     "empty props",
			prefix:   "foo",
			props:    &MockProps{},
			expected: "foo-08f44b07",
		},
		{
			name:   "full props",
			prefix: "foo",
			props: &MockProps{
				String: dry.ToPtr("foo-bar-baz"),
				Int:    dry.ToPtr[int](42),
				Bool:   dry.ToPtr[bool](true),
				Map:    dry.ToPtr(map[string]*string{"foo": dry.ToPtr("bar")}),
			},
			expected: "foo-69975559",
		},
		{
			name:   "FuzzResourceID/seed#0",
			prefix: "foo",
			props: &MockProps{
				String: dry.ToPtr("foo-bar-baz"),
			},
			expected: "foo-19922604",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			is := assert.New(t)
			got := ResourceID(tc.prefix, tc.props)
			is.Equal(tc.expected, *got)
		})
	}
}

func TestExtractResourceID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		resource string
		want     string
	}{
		{
			resource: "foo-19922604",
			want:     "foo",
		},
		{
			resource: "foo-19922604-12345678",
			want:     "foo-19922604",
		},
		{
			resource: "foo-abcdefgh",
			want:     "foo-abcdefgh",
		},
		{
			resource: "foo-bar-baz",
			want:     "foo-bar-baz",
		},
		{
			resource: "Resource",
			want:     "Resource",
		},
		{
			resource: "Default",
			want:     "Default",
		},
		{
			resource: "",
			want:     "unknown",
		},
	}
	for _, tc := range tests {
		t.Run(tc.resource, func(t *testing.T) {
			t.Parallel()
			is := assert.New(t)
			got := ExtractResource(dry.ToPtr(tc.resource))
			is.Equal(tc.want, got)
		})
	}
}

func FuzzResourceID(f *testing.F) {
	f.Add("foo", []byte(`{"string": "foo-bar-baz"}`))
	f.Add("", []byte(`{}`))
	f.Add("bar", []byte(`{"int": 42}`))
	f.Add("baz", []byte(`{"bool": true}`))
	f.Add("qux", []byte(`{"map": {"foo": "bar"}}`))
	f.Add("quux", []byte(`{"string": "foo-bar-baz", "int": 42, "bool": true, "map": {"foo": "bar"}}`))

	f.Fuzz(func(t *testing.T, prefix string, propBytes []byte) {
		props := new(MockProps)
		if err := json.Unmarshal(propBytes, props); err != nil {
			// Ignore the error if the props are not valid.
			// This is expected to happen in the fuzzing test.
			return
		}
		if prefix == "" {
			// If the prefix is empty, set the prefix to unknown.
			prefix = "unknown"
		}

		is := assert.New(t)
		got := *ResourceID(prefix, props)
		require.NotEmpty(t, got)
		parts := strings.Split(got, "-")
		require.GreaterOrEqual(t, len(parts), 2)

		// Check that the sum of the first part is equal to the prefix.
		is.Equal(strings.Join(parts[:len(parts)-1], "-"), prefix)
		// Check that the last part is a valid hash.
		is.Len(parts[len(parts)-1], 8)
		is.Regexp("^[a-f0-9]{8}$", parts[len(parts)-1])
	})
}
