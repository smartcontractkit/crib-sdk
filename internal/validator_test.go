package internal

import (
	"reflect"
	"testing"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockFieldLevel implements validator.FieldLevel for testing
type mockFieldLevel struct {
	field reflect.Value
}

func (m *mockFieldLevel) Top() reflect.Value      { return reflect.Value{} }
func (m *mockFieldLevel) Parent() reflect.Value   { return reflect.Value{} }
func (m *mockFieldLevel) Field() reflect.Value    { return m.field }
func (m *mockFieldLevel) FieldName() string       { return "testField" }
func (m *mockFieldLevel) StructFieldName() string { return "TestField" }
func (m *mockFieldLevel) Param() string           { return "" }
func (m *mockFieldLevel) GetTag() string          { return "" }
func (m *mockFieldLevel) ExtractType(field reflect.Value) (reflect.Value, reflect.Kind, bool) {
	return field, field.Kind(), true
}

func (m *mockFieldLevel) GetStructFieldOK() (reflect.Value, reflect.Kind, bool) {
	return reflect.Value{}, reflect.Invalid, false
}

func (m *mockFieldLevel) GetStructFieldOKAdvanced(parent reflect.Value, namespace string) (reflect.Value, reflect.Kind, bool) {
	return reflect.Value{}, reflect.Invalid, false
}

func (m *mockFieldLevel) GetStructFieldOK2() (reflect.Value, reflect.Kind, bool, bool) {
	return reflect.Value{}, reflect.Invalid, false, false
}

func (m *mockFieldLevel) GetStructFieldOKAdvanced2(parent reflect.Value, namespace string) (reflect.Value, reflect.Kind, bool, bool) {
	return reflect.Value{}, reflect.Invalid, false, false
}

func TestValidator(t *testing.T) {
	t.Parallel()

	v, err := NewValidator()
	require.NoError(t, err, "Failed to create validator")

	tests := []struct {
		desc         string
		input        any
		validation   string
		errAssertion assert.ErrorAssertionFunc
	}{
		// version validation tests
		{
			desc:         "Valid Version",
			input:        "v1.2.3",
			validation:   "version",
			errAssertion: assert.NoError,
		},
		{
			desc:         "Alt-Valid Version",
			input:        "1.2.3",
			validation:   "version",
			errAssertion: assert.NoError,
		},
		{
			desc:         "Invalid Version",
			input:        gofakeit.Adjective(),
			validation:   "version",
			errAssertion: assert.Error,
		},

		// yaml validation tests
		{
			desc:         "Valid YAML",
			input:        map[string]any{"key": "value"},
			validation:   "yaml",
			errAssertion: assert.NoError,
		},
		{
			desc:         "Invalid YAML",
			input:        map[string]any{"key": make(chan int)}, // Invalid YAML due to channel type
			validation:   "yaml",
			errAssertion: assert.Error,
		},
		// exclusive_of validation tests
		{
			desc: "Valid exclusive_of",
			input: struct {
				Key1 string `validate:"exclusive_of=Key2"`
				Key2 string `validate:"exclusive_of=Key1"`
			}{
				Key1: "value1",
			},
			validation:   "exclusive_of=Key2",
			errAssertion: assert.NoError,
		},
		{
			desc: "Invalid exclusive_of",
			input: struct {
				Key1 string `validate:"exclusive_of=Key2"`
				Key2 string `validate:"exclusive_of=Key1"`
			}{
				Key1: "value1",
				Key2: "value2",
			},
			validation:   "exclusive_of=Key2",
			errAssertion: assert.Error,
		},
		{
			desc: "Valid exclusive_of with interface",
			input: struct {
				Key1 string                      `validate:"exclusive_of=Key2"`
				Key2 interface{ Error() string } `validate:"exclusive_of=Key1"`
			}{
				Key1: "value1",
			},
			validation:   "exclusive_of=Key2",
			errAssertion: assert.NoError,
		},
		{
			desc: "Invalid exclusive_of with interface",
			input: struct {
				Key1 string                      `validate:"exclusive_of=Key2"`
				Key2 interface{ Error() string } `validate:"exclusive_of=Key1"`
			}{
				Key1: "value1",
				Key2: assert.AnError, // Using an error type to simulate an interface
			},
			validation:   "exclusive_of=Key2",
			errAssertion: assert.Error,
		},
		{
			desc: "Valid exclusive_of all empty",
			input: struct {
				Key1 string `validate:"exclusive_of=Key2"`
				Key2 string `validate:"exclusive_of=Key1"`
			}{},
			validation:   "exclusive_of=Key2",
			errAssertion: assert.NoError,
		},
		// ExprLang validation tests
		{
			desc:         "Valid ExprLang",
			input:        "2 + 2 == 4",
			validation:   "expr",
			errAssertion: assert.NoError,
		},
		{
			desc:         "Valid ExprLang with variables",
			input:        "let v = '3.40.0'; v >= '3.40.0'",
			validation:   "expr",
			errAssertion: assert.NoError,
		},
		{
			desc:         "Invalid ExprLang",
			input:        "2 +",
			validation:   "expr",
			errAssertion: assert.Error,
		},
	}
	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			t.Parallel()

			var err error
			require.NotPanics(t, func() {
				err = v.Var(tc.input, tc.validation)
			})
			tc.errAssertion(t, err, "Validation failed for input: %v with validation: %s", tc.input, tc.validation)
			if err != nil {
				assert.ErrorAs(t, err, &validator.ValidationErrors{}, "Expected validation error for input: %v with validation: %s", tc.input, tc.validation)
			}
		})
	}
}

func TestValidateImageURI(t *testing.T) {
	tests := []struct {
		name     string
		imageURI string
		want     bool
	}{
		// Valid cases
		{
			name:     "simple image name",
			imageURI: "nginx",
			want:     true,
		},
		{
			name:     "image with tag",
			imageURI: "nginx:latest",
			want:     true,
		},
		{
			name:     "image with version tag",
			imageURI: "nginx:1.21.0",
			want:     true,
		},
		{
			name:     "image with namespace",
			imageURI: "library/nginx:latest",
			want:     true,
		},
		{
			name:     "gcr.io registry",
			imageURI: "gcr.io/my-project/my-image:v1.0.0",
			want:     true,
		},
		{
			name:     "k8s registry",
			imageURI: "registry.k8s.io/kube-apiserver:v1.28.0",
			want:     true,
		},
		{
			name:     "docker hub with namespace",
			imageURI: "docker.io/library/nginx:latest",
			want:     true,
		},
		{
			name:     "registry with port",
			imageURI: "localhost:5000/my-image:latest",
			want:     true,
		},
		{
			name:     "registry with port and namespace",
			imageURI: "my-registry.com:5000/my-namespace/my-image:v1.0.0",
			want:     true,
		},
		{
			name:     "image with sha256 digest",
			imageURI: "nginx@sha256:1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
			want:     true,
		},
		{
			name:     "registry with digest",
			imageURI: "gcr.io/my-project/my-image@sha256:1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
			want:     true,
		},
		{
			name:     "registry with port and digest",
			imageURI: "localhost:5000/my-image@sha256:1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
			want:     true,
		},
		{
			name:     "sha512 digest",
			imageURI: "nginx@sha512:1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
			want:     true,
		},
		{
			name:     "sha1 digest",
			imageURI: "nginx@sha1:1234567890abcdef12345678",
			want:     true,
		},
		{
			name:     "complex nested namespace",
			imageURI: "registry.example.com/team/project/service:v1.2.3",
			want:     true,
		},
		{
			name:     "image with underscores",
			imageURI: "my_registry.com/my_namespace/my_image:my_tag",
			want:     true,
		},
		{
			name:     "image with dots in tag",
			imageURI: "nginx:1.21.0-alpine",
			want:     true,
		},

		// Invalid cases
		{
			name:     "empty string",
			imageURI: "",
			want:     false,
		},
		{
			name:     "whitespace only",
			imageURI: "   ",
			want:     false,
		},
		{
			name:     "contains space",
			imageURI: "nginx latest",
			want:     false,
		},
		{
			name:     "contains tab",
			imageURI: "nginx\tlatest",
			want:     false,
		},
		{
			name:     "contains newline",
			imageURI: "nginx\nlatest",
			want:     false,
		},
		{
			name:     "multiple @ symbols",
			imageURI: "nginx@sha256:abc@def",
			want:     false,
		},
		{
			name:     "invalid digest format - no colon",
			imageURI: "nginx@sha256abc",
			want:     false,
		},
		{
			name:     "invalid digest format - empty algorithm",
			imageURI: "nginx@:abc123",
			want:     false,
		},
		{
			name:     "invalid digest format - empty hash",
			imageURI: "nginx@sha256:",
			want:     false,
		},
		{
			name:     "invalid digest algorithm",
			imageURI: "nginx@md5:abc123",
			want:     false,
		},
		{
			name:     "consecutive slashes",
			imageURI: "registry.com//namespace/image:tag",
			want:     false,
		},
		{
			name:     "consecutive dots",
			imageURI: "registry..com/namespace/image:tag",
			want:     false,
		},
		{
			name:     "empty path component",
			imageURI: "registry.com//image:tag",
			want:     false,
		},
		{
			name:     "registry starts with hyphen",
			imageURI: "-registry.com/image:tag",
			want:     false,
		},
		{
			name:     "registry ends with hyphen",
			imageURI: "registry-.com/image:tag",
			want:     false,
		},
		{
			name:     "registry starts with dot",
			imageURI: ".registry.com/image:tag",
			want:     false,
		},
		{
			name:     "registry ends with dot",
			imageURI: "registry.com./image:tag",
			want:     false,
		},
		{
			name:     "invalid characters in registry",
			imageURI: "registry@.com/image:tag",
			want:     false,
		},
		{
			name:     "repository component starts with hyphen",
			imageURI: "registry.com/-namespace/image:tag",
			want:     false,
		},
		{
			name:     "repository component ends with hyphen",
			imageURI: "registry.com/namespace-/image:tag",
			want:     false,
		},
		{
			name:     "tag starts with hyphen",
			imageURI: "nginx:-latest",
			want:     false,
		},
		{
			name:     "tag starts with dot",
			imageURI: "nginx:.latest",
			want:     false,
		},
		{
			name:     "invalid characters in tag",
			imageURI: "nginx:latest@",
			want:     false,
		},
		{
			name:     "repository component too long",
			imageURI: "nginx/" + string(make([]byte, 64)) + ":latest",
			want:     false,
		},
		{
			name:     "tag too long",
			imageURI: "nginx:" + string(make([]byte, 129)),
			want:     false,
		},
		{
			name:     "registry name too long",
			imageURI: string(make([]byte, 254)) + "/nginx:latest",
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			field := reflect.ValueOf(tt.imageURI)
			fl := &mockFieldLevel{field: field}

			got := validateImageURI(fl)
			if got != tt.want {
				t.Errorf("validateImageURI() = %v, want %v for input: %q", got, tt.want, tt.imageURI)
			}
		})
	}
}

func TestValidateImageURI_NonStringField(t *testing.T) {
	// Test with non-string field
	field := reflect.ValueOf(123)
	fl := &mockFieldLevel{field: field}

	got := validateImageURI(fl)
	if got != false {
		t.Errorf("validateImageURI() = %v, want false for non-string field", got)
	}
}

func TestIsValidRegistryName(t *testing.T) {
	tests := []struct {
		name     string
		registry string
		want     bool
	}{
		{"valid simple", "registry.com", true},
		{"valid with subdomain", "my.registry.com", true},
		{"valid localhost", "localhost", true},
		{"valid with numbers", "registry123.com", true},
		{"valid with hyphens", "my-registry.com", true},
		{"empty string", "", false},
		{"starts with hyphen", "-registry.com", false},
		{"ends with hyphen", "registry.com-", false},
		{"starts with dot", ".registry.com", false},
		{"ends with dot", "registry.com.", false},
		{"too long", string(make([]byte, 254)), false},
		{"invalid characters", "registry@.com", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isValidRegistryName(tt.registry)
			if got != tt.want {
				t.Errorf("isValidRegistryName() = %v, want %v for input: %q", got, tt.want, tt.registry)
			}
		})
	}
}

func TestIsValidPort(t *testing.T) {
	tests := []struct {
		name string
		port string
		want bool
	}{
		{"valid port", "5000", true},
		{"valid port 80", "80", true},
		{"valid port 443", "443", true},
		{"valid port 8080", "8080", true},
		{"empty string", "", false},
		{"non-numeric", "abc", false},
		{"too long", "123456", false},
		{"mixed characters", "80abc", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isValidPort(tt.port)
			if got != tt.want {
				t.Errorf("isValidPort() = %v, want %v for input: %q", got, tt.want, tt.port)
			}
		})
	}
}

func TestIsValidRepositoryComponent(t *testing.T) {
	tests := []struct {
		name      string
		component string
		want      bool
	}{
		{"valid simple", "nginx", true},
		{"valid with hyphen", "my-image", true},
		{"valid with underscore", "my_image", true},
		{"valid with dot", "my.image", true},
		{"valid with numbers", "image123", true},
		{"empty string", "", false},
		{"starts with hyphen", "-image", false},
		{"ends with hyphen", "image-", false},
		{"starts with dot", ".image", false},
		{"ends with dot", "image.", false},
		{"starts with underscore", "_image", false},
		{"ends with underscore", "image_", false},
		{"too long", string(make([]byte, 64)), false},
		{"invalid characters", "image@", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isValidRepositoryComponent(tt.component)
			if got != tt.want {
				t.Errorf("isValidRepositoryComponent() = %v, want %v for input: %q", got, tt.want, tt.component)
			}
		})
	}
}

func TestIsValidTag(t *testing.T) {
	tests := []struct {
		name string
		tag  string
		want bool
	}{
		{"valid simple", "latest", true},
		{"valid with version", "v1.2.3", true},
		{"valid with hyphen", "my-tag", true},
		{"valid with underscore", "my_tag", true},
		{"valid with dot", "1.2.3", true},
		{"valid complex", "v1.2.3-alpha.1", true},
		{"empty string", "", false},
		{"starts with hyphen", "-tag", false},
		{"starts with dot", ".tag", false},
		{"too long", string(make([]byte, 129)), false},
		{"invalid characters", "tag@", false},
		{"invalid characters space", "tag latest", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isValidTag(tt.tag)
			if got != tt.want {
				t.Errorf("isValidTag() = %v, want %v for input: %q", got, tt.want, tt.tag)
			}
		})
	}
}

func TestIsAlphaNumeric(t *testing.T) {
	tests := []struct {
		name string
		char byte
		want bool
	}{
		{"lowercase letter", 'a', true},
		{"uppercase letter", 'A', true},
		{"digit", '5', true},
		{"hyphen", '-', false},
		{"underscore", '_', false},
		{"dot", '.', false},
		{"space", ' ', false},
		{"special char", '@', false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isAlphaNumeric(tt.char)
			if got != tt.want {
				t.Errorf("isAlphaNumeric() = %v, want %v for input: %c", got, tt.want, tt.char)
			}
		})
	}
}

// Integration test with actual validator
func TestValidateImageURIIntegration(t *testing.T) {
	subject, err := NewValidator()
	if err != nil {
		t.Fatalf("Failed to create validator: %v", err)
	}

	type TestStruct struct {
		ImageURI string `validate:"image_uri"`
	}

	tests := []struct {
		name     string
		imageURI string
		wantErr  bool
	}{
		{"valid image", "nginx:latest", false},
		{"valid with registry", "gcr.io/project/image:v1.0.0", false},
		{"invalid empty", "", true},
		{"invalid with space", "nginx latest", true},
		{"invalid digest", "nginx@invalid", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := TestStruct{ImageURI: tt.imageURI}
			err := subject.Struct(&s)

			if tt.wantErr && err == nil {
				t.Errorf("Expected validation error for %q, but got none", tt.imageURI)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("Unexpected validation error for %q: %v", tt.imageURI, err)
			}
		})
	}
}
