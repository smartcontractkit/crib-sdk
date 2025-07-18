package crib

import "context"

// Props is an interface that all components should accept as an argument.
// It can be used to expand arguments, provide validation, provide logging, etc.
type Props interface {
	// Validate validates the props and returns an error of all, if any, validation fails.
	Validate(ctx context.Context) error
}
