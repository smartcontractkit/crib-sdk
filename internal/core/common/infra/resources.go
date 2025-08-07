package infra

import (
	"cmp"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"slices"
	"strings"

	"github.com/davecgh/go-spew/spew"

	"github.com/smartcontractkit/crib-sdk/internal/adapter/mempools"
	"github.com/smartcontractkit/crib-sdk/internal/core/common/dry"
	"github.com/smartcontractkit/crib-sdk/internal/core/domain"
)

// ToRFC1123 generates a unique and stable name compatible with Kubernetes DNS-1035 labels.
//
// DNS-1035 labels must match the regex '[a-z]([-a-z0-9]*[a-z0-9])?' and be 63 characters or less.
// This means:
// - Must start with a lowercase letter
// - Must end with a lowercase letter or digit
// - May contain lowercase letters, digits, and hyphens in between
// - Must be 63 characters or less
//
// The generated name will have the form:
//
//	<comp0>-<comp1>-..<compN>-<short-hash>
//
// Where <comp> are the path components (assuming they are separated by "/").
//
// Note that if the total length is longer than 63 characters, we will trim
// the first components since the last components usually encode more meaning.
//
// As a caveat, if the id is trimmed down to 63 characters, this method will prepend
// "id-" to the start of the result to ensure that it starts with a letter.
func ToRFC1123(id string) string {
	id = strings.TrimSpace(id)
	if id == "" {
		return domain.CDK8sUnknown
	}
	// Don't touch the resource id if it matches a cdk8s magic string.
	if slices.Contains([]string{domain.CDK8sResource, domain.CDK8sDefault}, id) {
		return strings.ToLower(id)
	}

	// Convert to lowercase for DNS-1035 compliance
	result := strings.ToLower(id)

	// Replace invalid characters with hyphens
	// Only allow lowercase letters, digits, and hyphens
	result = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			return r
		}
		return '-'
	}, result)

	// Remove leading and trailing hyphens
	result = strings.Trim(result, "-")

	// Remove consecutive hyphens.
	for strings.Contains(result, "--") {
		result = strings.ReplaceAll(result, "--", "-")
	}

	// If the result is empty after trimming, return "unknown".
	if result == "" {
		return domain.CDK8sUnknown
	}

	// Split the result by hyphens and assume that the first parts make up the prefix.
	parts := strings.Split(result, "-")
	if len(parts) == 1 && isHashHex(parts[0]) {
		parts = append([]string{domain.CDK8sUnknown}, parts...)
	}
	prefix := strings.Join(parts[:len(parts)-1], "-") // All but the last part
	hash := parts[len(parts)-1]                       // This might be the hash or part of the prefix
	// Determine if the last part is a valid hash. If it is not, we will treat it as part of the prefix.
	if !isHashHex(hash) {
		// Append a hash to the end of the prefix, and unset the hash.
		prefix += "-" + hash
		hash = ""
	}

	// Ensure the prefix starts with a letter (DNS-1035 requirement)
	if prefix != "" && !isLowercaseLetter(rune(prefix[0])) {
		// Find the first letter in the string
		firstLetterIndex := -1
		for i, r := range prefix {
			if isLowercaseLetter(r) {
				firstLetterIndex = i
				break
			}
		}

		if firstLetterIndex == -1 {
			// No letters found, prepend with "unknown" and hash the prefix
			prefix = domain.CDK8sUnknown + "-" + fnvHash([]byte(prefix))
		} else {
			// Start from the first letter
			prefix = prefix[firstLetterIndex:]
		}
	}

	// Rejoin the prefix and hash.
	result = prefix + "-" + hash
	// Remove leading and trailing hyphens one more time.
	result = strings.Trim(result, "-")

	// Ensure the hash ends with alphanumeric (DNS-1035 requirement).
	// If it doesn't, append `-crib` to the end.
	// Theoretically, the hash should always be alphanumeric, but this is a safeguard.
	if hash != "" && !isAlphanumericLowercase(rune(hash[len(hash)-1])) {
		hash += "-crib" //nolint:ineffassign // Append a suffix to ensure it ends with an alphanumeric character.
	}

	if len(result) <= 63 {
		// If the result is already within the 63 character limit, return it.
		return result
	}

	// The total length exceeds 63 characters, we need to trim the result, starting
	// from the beginning, to ensure the total length is 63 characters or fewer.

	// Trim from the start, keeping the last 63 characters.
	result = result[len(result)-63:]

	// Ensure it is still valid after trimming. If it is not valid,
	// we will prepend "id-" to the start of the result and trim it again.
	// This is to ensure that the result is always a valid DNS-1035 label.
	// It avoids cases where a resource id is originally valid, but after trimming,
	// will never be valid - looping over and over checking the result.
	if result != "" && !isLowercaseLetter(rune(result[0])) {
		// Prepend "id-" to the result to ensure it starts with a letter and
		// trim the first three characters again to stay within the 63 character limit.
		result = "id-" + result[3:]
	}
	return result
}

// isLowercaseLetter checks if a rune is a lowercase letter.
func isLowercaseLetter(r rune) bool {
	return r >= 'a' && r <= 'z'
}

// isAlphanumericLowercase checks if a rune is a lowercase letter or digit.
func isAlphanumericLowercase(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
}

// ResourceID generates a resource id for the given prefix and props.
func ResourceID(prefix string, props any) *string {
	prefix = strings.TrimSpace(prefix)
	// Don't touch the resource id if it matches a cdk8s magic string.
	if cmp.Or(prefix == domain.CDK8sResource, prefix == domain.CDK8sDefault) {
		return dry.ToPtr(prefix)
	}
	if prefix == "" {
		// If the prefix is empty, set the prefix to unknown.
		prefix = domain.CDK8sUnknown
	}

	// Encode the props into a byte slice.
	b := encode(props)
	// Get the hash as a string and truncate it to 8 characters.
	hashed := fnvHash(b)[:8]
	id := fmt.Sprintf("%s-%s", prefix, hashed)
	return dry.ToPtr(id)
}

// encode attempts to encode the given value into a byte slice. It will attempt various encoding methods
// until one succeeds. If no encoding method succeeds, it will panic.
func encode(v any) []byte {
	// If the value is a string, return it as a byte slice.
	if v, ok := v.(string); ok {
		return []byte(v)
	}
	// If the value is a byte slice, return it as is.
	if v, ok := v.([]byte); ok {
		return v
	}

	// Attempt the various marshalers.
	buf, ret := mempools.BytesBuffer.Get()
	defer ret()

	// Attempt to JSON marshal the value.
	if err := json.NewEncoder(buf).Encode(v); err == nil {
		return buf.Bytes()
	}
	buf.Reset() // Reset the buffer for the next attempt.
	// Attempt to Gob encode the value.
	if err := gob.NewEncoder(buf).Encode(v); err == nil {
		return buf.Bytes()
	}
	buf.Reset() // Reset the buffer for the next attempt.

	// If that fails, try to call the String() method on the value.
	if strer, ok := v.(fmt.Stringer); ok {
		if str := strer.String(); str != "" {
			return []byte(str)
		}
	}

	// Finally, attempt to spew the value to a byte slice.
	if spew.Fdump(buf, v); buf.Len() > 0 { //nolint:gocritic // Inline is easier to reason about.
		return buf.Bytes()
	}

	// If all attempts fail, panic with a message indicating the type of the value.
	panic(fmt.Sprintf("Could not encode value of type %T: %#v", v, v))
}

// ExtractResource extracts the resource name from the given generated resource id.
// This method is exposed primarily for testing purposes.
func ExtractResource(id *string) string {
	if id == nil || *id == "" {
		return domain.CDK8sUnknown
	}
	str := *id
	parts := strings.Split(str, "-")
	if len(parts) == 1 {
		return str
	}
	// Check if the last part is an 8 character hex string. If it is not, return the full ID.
	if !isHashHex(parts[len(parts)-1]) {
		return str
	}
	// Otherwise return the first part, concatenated with the rest of the parts, up to the last part.
	return strings.Join(parts[:len(parts)-1], "-")
}

func isHashHex(s string) bool {
	match := func(c rune) bool {
		return (c < '0' || c > '9') && (c < 'a' || c > 'f')
	}
	if len(s) != 8 || strings.IndexFunc(s, match) != -1 {
		return false
	}
	return true
}

// fnvHash computes the FNV hash of the given byte slice and returns it as a hex string.
func fnvHash(b []byte) string {
	h := fnv.New64a()
	_, _ = h.Write(b)
	hashed := h.Sum(nil)
	return fmt.Sprintf("%x", hashed)
}
