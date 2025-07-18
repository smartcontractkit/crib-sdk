package internal

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/creasty/defaults"
	"github.com/expr-lang/expr"
	"github.com/go-playground/validator/v10"
	"golang.org/x/mod/semver"
	"gopkg.in/yaml.v3"

	"github.com/smartcontractkit/crib-sdk/internal/core/common/dry"
)

var instance = sync.OnceValues(func() (*Validator, error) {
	v := validator.New(validator.WithRequiredStructEnabled())
	// Here you can register custom validation rules.
	errs := errors.Join(
		v.RegisterValidation("valid_chain_selector", validateChainSelector),
		v.RegisterValidation("version", validateSemver),
		v.RegisterValidation("yaml", validateYAML),
		v.RegisterValidation("exclusive_of", exclusiveOf, true),
		v.RegisterValidation("expr", validateExprLang),
		v.RegisterValidation("image_uri", validateImageURI),
	)
	return dry.Wrapf2(&Validator{Validate: v}, errs, "failed to initialize validator")
})

type Validator struct {
	*validator.Validate
}

// Struct initializes default values for a struct and validates a structs exposed fields, and automatically validates
// nested structs, unless otherwise specified.
//
// Default values initializes members in a struct referenced by a pointer.
// Maps and slices are initialized by `make` and other primitive types are set with default values.
// `ptr` should be a struct pointer.
//
// If the passed in value implements the interface:
//
//	type Setter interface {
//		SetDefaults()
//	}
//
// It will be called before validation. This is useful for setting defaults that are not
// handled by the `defaults` package, such as very complex values.
//
// It returns InvalidValidationError for bad values passed in and nil or ValidationErrors as error otherwise.
// ou will need to assert the error if it's not nil eg. err.(validator.ValidationErrors) to access the array of errors.
func (v *Validator) Struct(i any) error {
	isNil := i == nil || reflect.ValueOf(i).IsNil()
	if !isNil {
		if err := defaults.Set(i); err != nil {
			return fmt.Errorf("setting defaults: %w", err)
		}
	}

	return v.Validate.Struct(i)
}

// CanUpdate returns true when the given value is an initial value of its type. It indicates
// to the defaults handler that this value is safe to manipulate. This method should be called
// within a SetDefaults function call on types that satisfy the Setter interface.
func CanUpdate(i any) bool {
	return defaults.CanUpdate(i)
}

// NewValidator creates a new validator instance with custom validation rules.
func NewValidator() (*Validator, error) {
	return instance()
}

func validateChainSelector(f validator.FieldLevel) bool {
	return true // Implement your validation logic here
}

// validateImageURI validates that a field contains a valid Kubernetes image URI.
// A valid image URI follows the format: [registry[:port]/]namespace/name[:tag|@digest]
// Examples:
//   - nginx:latest
//   - gcr.io/my-project/my-image:v1.0.0
//   - registry.k8s.io/kube-apiserver:v1.28.0
//   - my-registry.com:5000/my-namespace/my-image@sha256:abc123...
func validateImageURI(fl validator.FieldLevel) bool {
	if fl.Field().Kind() != reflect.String {
		return false // Only string fields can be validated as image URIs
	}

	imageURI := strings.TrimSpace(fl.Field().String())
	if imageURI == "" {
		return false // Empty string is not a valid image URI
	}

	// Check for invalid characters that are not allowed in image URIs
	if strings.ContainsAny(imageURI, " \t\n\r") {
		return false // Whitespace characters are not allowed
	}

	// Split by '@' to separate digest if present
	nameAndTag, hasDigest := extractNameAndValidateDigest(imageURI)
	if hasDigest && nameAndTag == "" {
		return false // Invalid digest format
	}

	// Split by ':' to separate tag if present (and no digest)
	imageName, tag := parseImageNameAndTag(imageURI, nameAndTag)

	// Validate image name format
	if imageName == "" {
		return false
	}

	// Check for consecutive separators
	if strings.Contains(imageName, "//") || strings.Contains(imageName, "..") {
		return false
	}

	// Split into registry and repository parts
	parts := strings.Split(imageName, "/")

	// Validate each part
	for i, part := range parts {
		if part == "" {
			return false // Empty parts are not allowed
		}

		// Replace the if-else chain with a switch statement
		switch {
		case i == 0 && strings.Contains(part, ":"):
			// First part might be registry with port
			registryParts := strings.Split(part, ":")
			if len(registryParts) != 2 {
				return false
			}
			// Validate registry name and port
			if !isValidRegistryName(registryParts[0]) {
				return false
			}
			if !isValidPort(registryParts[1]) {
				return false
			}
		case i == 0 && len(parts) > 1 && strings.Contains(part, "."):
			// First part looks like a registry (contains dots and has more parts after it)
			if !isValidRegistryName(part) {
				return false
			}
		default:
			// Validate repository name component
			if !isValidRepositoryComponent(part) {
				return false
			}
		}
	}

	// Validate tag if present
	if tag != "" && !isValidTag(tag) {
		return false
	}

	return true
}

// extractNameAndValidateDigest separates digest from image URI and validates it.
func extractNameAndValidateDigest(imageURI string) (string, bool) {
	if !strings.Contains(imageURI, "@") {
		return imageURI, false
	}

	parts := strings.Split(imageURI, "@")
	if len(parts) != 2 {
		return "", true // Invalid: multiple '@' symbols
	}

	nameAndTag := parts[0]
	digest := parts[1]

	// Validate digest format (should start with algorithm:hash)
	if !strings.Contains(digest, ":") {
		return "", true // Invalid: no colon in digest
	}

	digestParts := strings.SplitN(digest, ":", 2)
	if len(digestParts) != 2 || digestParts[0] == "" || digestParts[1] == "" {
		return "", true // Invalid: empty algorithm or hash
	}

	// Common digest algorithms
	validAlgorithms := map[string]bool{
		"sha256": true,
		"sha512": true,
		"sha1":   true,
	}
	if !validAlgorithms[digestParts[0]] {
		return "", true // Invalid: unsupported algorithm
	}

	return nameAndTag, true
}

// parseImageNameAndTag separates image name from tag.
func parseImageNameAndTag(imageURI, nameAndTag string) (imageName, tag string) {
	if strings.Contains(imageURI, "@") || !strings.Contains(nameAndTag, ":") {
		return nameAndTag, ""
	}

	// Find the last ':' to handle registry:port/image:tag format
	lastColon := strings.LastIndex(nameAndTag, ":")
	imageName = nameAndTag[:lastColon]
	tag = nameAndTag[lastColon+1:]

	// Check if this might be a port number instead of a tag
	// If the part before ':' doesn't contain '/', it's likely registry:port
	if !strings.Contains(imageName, "/") && isLikelyPort(tag) {
		// Looks like a port, treat the whole thing as image name
		return nameAndTag, ""
	}

	return imageName, tag
}

// isLikelyPort checks if a string looks like a port number.
func isLikelyPort(s string) bool {
	if s == "" || len(s) > 5 {
		return false
	}

	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// isValidRegistryName validates registry hostname.
func isValidRegistryName(registry string) bool {
	if registry == "" || len(registry) > 253 {
		return false
	}

	// Must not start or end with hyphen or dot
	if registry[0] == '-' || registry[0] == '.' ||
		registry[len(registry)-1] == '-' || registry[len(registry)-1] == '.' {
		return false
	}

	// Check for valid characters (alphanumeric, hyphen, dot, underscore)
	for _, r := range registry {
		if !isAlphaNumeric(byte(r)) && r != '-' && r != '.' && r != '_' {
			return false
		}
	}

	// Additional check: validate each domain component separately
	parts := strings.Split(registry, ".")
	for _, part := range parts {
		if part == "" {
			return false // Empty parts not allowed
		}
		// Each part must not start or end with hyphen
		if part[0] == '-' || part[len(part)-1] == '-' {
			return false
		}
		// Each part must be alphanumeric with hyphens/underscores only in middle
		for i, r := range part {
			if !isAlphaNumeric(byte(r)) && r != '-' && r != '_' {
				return false
			}
			// Hyphens only allowed in middle positions (underscores can be anywhere)
			if r == '-' && (i == 0 || i == len(part)-1) {
				return false
			}
		}
	}

	return true
}

// isValidPort validates port number.
func isValidPort(port string) bool {
	if port == "" {
		return false
	}

	// Must be numeric and within valid range
	for _, r := range port {
		if r < '0' || r > '9' {
			return false
		}
	}

	// Port must be between 1 and 65535
	if len(port) > 5 {
		return false
	}

	return true
}

// isValidRepositoryComponent validates a single component of repository name.
func isValidRepositoryComponent(component string) bool {
	if component == "" || len(component) > 63 {
		return false
	}

	// Must start and end with alphanumeric
	if !isAlphaNumeric(component[0]) || !isAlphaNumeric(component[len(component)-1]) {
		return false
	}

	// Check for valid characters (alphanumeric, hyphen, underscore, dot)
	for _, r := range component {
		if !isAlphaNumeric(byte(r)) && r != '-' && r != '_' && r != '.' {
			return false
		}
	}

	return true
}

// isValidTag validates image tag.
func isValidTag(tag string) bool {
	if tag == "" || len(tag) > 128 {
		return false
	}

	// Must not start with '-' or '.'
	if tag[0] == '-' || tag[0] == '.' {
		return false
	}

	// Check for valid characters (alphanumeric, hyphen, underscore, dot)
	for _, r := range tag {
		if !isAlphaNumeric(byte(r)) && r != '-' && r != '_' && r != '.' {
			return false
		}
	}

	return true
}

// isAlphaNumeric checks if character is alphanumeric.
func isAlphaNumeric(r byte) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')
}

func validateSemver(f validator.FieldLevel) bool {
	val := f.Field().String()
	if val == "" {
		return false // Empty string is not a valid version.
	}
	// Ensure the version starts with 'v' for semver validation.
	// Helm charts typically omit the 'v' prefix, but semver requires it.
	if val[0] != 'v' {
		// Add 'v' prefix if not present
		val = "v" + val
	}
	return semver.IsValid(val)
}

// validateYAML validates that a field contains valid YAML.
func validateYAML(f validator.FieldLevel) bool {
	defer func() {
		if r := recover(); r != nil {
			// If we panic, we assume the YAML is invalid.
			return
		}
	}()

	val, ok := f.Field().Interface().(map[string]any)
	if !ok {
		return false // Not a map, cannot validate as YAML.
	}

	// Encode the map to YAML and then decode it back to check for validity.
	yamlData, err := yaml.Marshal(val)
	if err != nil {
		return false // Failed to marshal to YAML, hence invalid.
	}

	// Return true if unmarshalling was successful, indicating valid YAML.
	var decoded map[string]any
	return yaml.Unmarshal(yamlData, &decoded) == nil
}

// exclusiveOf is the validation function
// The field under validation must not be present or is empty when any of the other specified fields are not present.
func exclusiveOf(fl validator.FieldLevel) bool {
	params := strings.Split(fl.Param(), ",")
	parent := fl.Parent()
	thisSet := hasValue(fl)

	// If this field is not set, always valid
	if !thisSet {
		return true
	}

	for _, param := range params {
		param = strings.TrimSpace(param)
		if param == "" {
			continue
		}
		field, _, _, found := fl.GetStructFieldOKAdvanced2(parent, param)
		if !found {
			continue
		}
		// Check if the other field is set (non-zero)
		if field.IsValid() && !field.IsZero() {
			return false // Both fields are set, invalid
		}
	}
	return true
}

// hasValue is the validation function for validating if the current field's value is not the default static value.
func hasValue(fl validator.FieldLevel) bool {
	field := fl.Field()
	switch field.Kind() {
	case reflect.Slice, reflect.Map, reflect.Ptr, reflect.Interface, reflect.Chan, reflect.Func:
		return !field.IsNil()
	default:
		if fl.Field().Kind() == reflect.Ptr && field.Interface() != nil {
			return true
		}
		return field.IsValid() && !field.IsZero()
	}
}

func validateExprLang(fl validator.FieldLevel) bool {
	if fl.Field().Kind() != reflect.String {
		return false // Only string fields can be validated with expr-lang
	}

	val := fl.Field().String()
	// Fill in dummy data to compile the expression.
	// TODO: Make less brittle.
	if strings.Contains(val, "%s") {
		// If the expression contains a placeholder, replace it with a buzzword.
		// This is just for validation purposes.
		val = strings.ReplaceAll(val, "%s", gofakeit.BuzzWord())
	}
	_, err := expr.Compile(val)
	return err == nil // Return true if the expression is valid, false otherwise
}
