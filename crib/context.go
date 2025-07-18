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
