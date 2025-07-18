package crib

import (
	"context"

	"github.com/aws/constructs-go/constructs/v10"

	"github.com/smartcontractkit/crib-sdk/internal"
)

// ConstructFromContext retrieves the constructs.Construct from the context.
func ConstructFromContext(ctx context.Context) constructs.Construct {
	return internal.ConstructFromContext(ctx)
}

// ContextWithConstruct creates a new context with supplied constructs.Construct value.
func ContextWithConstruct(ctx context.Context, c constructs.Construct) context.Context {
	return internal.ContextWithConstruct(ctx, c)
}

// ValidatorFromContext retrieves the validator from the context. If one does not exist, it is created.
func ValidatorFromContext(ctx context.Context) *internal.Validator {
	return internal.ValidatorFromContext(ctx)
}

// ContextWithValidator creates a new context with the supplied validator value.
func ContextWithValidator(ctx context.Context, v *internal.Validator) context.Context {
	return internal.ContextWithValidator(ctx, v)
}
