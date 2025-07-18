package port

import "context"

// Validator represents a common interface for validating inputs throughout the SDK.
type Validator interface {
	// Validate validates the input context and returns an error if validation fails.
	Validate(ctx context.Context) error
}
