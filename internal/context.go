package internal

import (
	"context"

	"github.com/aws/constructs-go/constructs/v10"

	"github.com/smartcontractkit/crib-sdk/internal/core/common/dry"
)

type (
	constructKey struct{}
	validatorKey struct{}
)

// ConstructFromContext retrieves the constructs.Construct from the context.
func ConstructFromContext(ctx context.Context) constructs.Construct {
	if ctx == nil {
		return nil
	}
	return dry.MustAs[constructs.Construct](ctx.Value(constructKey{}))
}

// ContextWithConstruct creates a new context with supplied constructs.Construct value.
func ContextWithConstruct(ctx context.Context, c constructs.Construct) context.Context {
	if ctx == nil {
		return nil
	}
	return context.WithValue(ctx, constructKey{}, c)
}

// ValidatorFromContext retrieves the validator from the context. If one does not exist, it is created.
func ValidatorFromContext(ctx context.Context) *Validator {
	if ctx == nil {
		return nil
	}
	v := dry.As[*Validator](ctx.Value(validatorKey{}))
	if v == nil {
		v, _ = NewValidator()
	}
	return v
}

// ContextWithValidator creates a new context with the supplied validator value.
func ContextWithValidator(ctx context.Context, v *Validator) context.Context {
	if ctx == nil {
		return nil
	}
	return context.WithValue(ctx, validatorKey{}, v)
}
