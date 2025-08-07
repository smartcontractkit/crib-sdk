package infra

import (
	"context"
	"encoding/gob"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/crib-sdk/internal/adapter/mempools"
	"github.com/smartcontractkit/crib-sdk/internal/core/common/dry"
	"github.com/smartcontractkit/crib-sdk/internal/core/domain"
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

// isDNS1035 checks if a string is DNS-1035 compliant
// DNS-1035 labels must match the regex '[a-z]([-a-z0-9]*[a-z0-9])?'
func isDNS1035(s string) bool {
	if len(s) == 0 || len(s) > 63 {
		return false
	}
	// Must start with lowercase letter
	if s[0] < 'a' || s[0] > 'z' {
		return false
	}
	// Must end with lowercase letter or digit
	lastChar := s[len(s)-1]
	if !((lastChar >= 'a' && lastChar <= 'z') || (lastChar >= '0' && lastChar <= '9')) {
		return false
	}
	// Check all characters are valid (lowercase letters, digits, hyphens)
	for _, r := range s {
		if !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-') {
			return false
		}
	}
	return true
}

func TestToRFC1123ID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple input",
			input:    "TestingApp",
			expected: "testingapp",
		},
		{
			name:     "valid input",
			input:    "foo-bar-123",
			expected: "foo-bar-123",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "unknown",
		},
		{
			name:     "with invalid characters",
			input:    "foo@bar#123",
			expected: "foo-bar-123",
		},
		{
			name:     "starts with non-alphanumeric",
			input:    "-foo-bar",
			expected: "foo-bar",
		},
		{
			name:     "ends with non-alphanumeric",
			input:    "foo-bar-",
			expected: "foo-bar",
		},
		{
			name:     "too long",
			input:    "very-long-name-that-exceeds-the-maximum-allowed-length-of-63-characters-and-should-be-truncated",
			expected: "maximum-allowed-length-of-63-characters-and-should-be-truncated",
		},
		{
			name:     "all invalid chars",
			input:    "@@##$$",
			expected: "unknown",
		},
		{
			name:     "mixed case with dots and underscores",
			input:    "Foo_Bar.123",
			expected: "foo-bar-123",
		},
		{
			name:     "uppercase input",
			input:    "FOO-BAR-123",
			expected: "foo-bar-123",
		},
		{
			name:     "starts with digit",
			input:    "123-foo-bar",
			expected: "foo-bar",
		},
		{
			name:     "consecutive special chars",
			input:    "foo@@##bar",
			expected: "foo-bar",
		},
		{
			name:     "only digits and hyphens",
			input:    "123-456-789",
			expected: "unknown-1aee501b712e8756",
		},
		{
			name:     "ends with hyphen after trimming",
			input:    "foo-bar-###",
			expected: "foo-bar",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			is := assert.New(t)
			got := ToRFC1123(tc.input)
			is.Equal(tc.expected, got)
			is.True(isDNS1035(got), "Result is not DNS-1035 compliant: %s", got)
		})
	}
}

func TestResourceID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		prefix   string
		props    any
		expected string
	}{
		{
			name:     "magic string Resource",
			prefix:   domain.CDK8sResource,
			expected: "Resource",
		},
		{
			name:     "magic string Default",
			prefix:   domain.CDK8sDefault,
			expected: "Default",
		},
		{
			name:     "real sdk id",
			prefix:   "sdk.Namespace",
			expected: "sdk.Namespace-5b9bc4ba",
		},
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
			is.LessOrEqual(len(*got), 63, "ID is longer than 63 characters: %s", *got)
		})
	}
}

func TestExtractResourceID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		resource string
		want     string
	}{
		{
			name:     "simple prefix without hash",
			resource: "TestingApp",
			want:     "TestingApp",
		},
		{
			name:     "simple prefix with hash",
			resource: "foo-19922604",
			want:     "foo",
		},
		{
			name:     "compound prefix with hash",
			resource: "foo-bar-baz-12345678",
			want:     "foo-bar-baz",
		},
		{
			name:     "invalid hash format",
			resource: "foo-abcdefgh",
			want:     "foo-abcdefgh",
		},
		{
			name:     "no hash separator",
			resource: "foo-bar-baz",
			want:     "foo-bar-baz",
		},
		{
			name:     "magic string Resource",
			resource: "Resource",
			want:     "Resource",
		},
		{
			name:     "magic string Default",
			resource: "Default",
			want:     "Default",
		},
		{
			name:     "empty string",
			resource: "",
			want:     "unknown",
		},
		{
			name:     "resource with special characters",
			resource: "foo-bar-baz-!@#$%^&*()_+",
			want:     "foo-bar-baz-!@#$%^&*()_+",
		},
		{
			name:     "valid sdk id with dot",
			resource: "sdk.Namespace",
			want:     "sdk.Namespace",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			is := assert.New(t)
			got := ExtractResource(dry.ToPtr(tc.resource))
			is.Equal(tc.want, got)
		})
	}
}

type jsonFixture struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
}

type gobFixture struct {
	MyStr string
	Ch    chan int
}

// stringerBad encodes to String() successfully, but is not encodable via JSON or gob
// because it contains a channel field.
type stringerBad struct{ C chan int }

func (s stringerBad) String() string { return "stringer-ok" }

// stringerEmpty hits the spew path because JSON and gob both fail, and String() returns empty.
type stringerEmpty struct{ C chan int }

func (s stringerEmpty) String() string { return "" }

func Test_encode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		desc string
		in   any
		want string
	}{
		{
			desc: "string",
			in:   "hello world",
			want: "hello world",
		},
		{
			desc: "bytes",
			in:   []byte("bytes here"),
			want: "bytes here",
		},
		{
			desc: "json",
			in:   jsonFixture{Name: "bob", Age: 2},
			want: `{"name":"bob","age":2}
`,
		},
		{
			desc: "gob",
			in:   &gobFixture{},
			want: func() string {
				buf, ret := mempools.BytesBuffer.Get()
				defer ret()
				err := gob.NewEncoder(buf).Encode(&gobFixture{})
				require.NoError(t, err, "gob encoding failed")
				return buf.String()
			}(),
		},
		{
			desc: "stringer",
			in:   stringerBad{C: make(chan int)},
			want: "stringer-ok",
		},
		{
			desc: "spew",
			in:   stringerEmpty{C: make(chan int)},
			want: "(infra.stringerEmpty) \n",
		},
	}

	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			t.Parallel()

			got := encode(tc.in)
			assert.Equal(t, tc.want, string(got))
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

		is := assert.New(t)
		got := *ResourceID(prefix, props)
		require.NotEmpty(t, got)
		parts := strings.Split(got, "-")
		require.GreaterOrEqualf(t, len(parts), 2, "ResourceID must have at least two parts: prefix and hash. Got: %s", got)

		// Check that the result is DNS-1035 compliant
		is.True(isDNS1035(got), "Result must be DNS-1035 compliant: %s", got)
		is.LessOrEqual(len(got), 63, "ID is longer than 63 characters: %s", got)
		// Check that the last part is a valid hash.
		is.Len(parts[len(parts)-1], 8)
		is.Regexp("^[a-f0-9]{8}$", parts[len(parts)-1])
	})
}
