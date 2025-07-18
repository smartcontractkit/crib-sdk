package crib

import (
	"context"

	"github.com/smartcontractkit/crib-sdk/internal/core/common/infra"
	"github.com/smartcontractkit/crib-sdk/internal/core/port"
)

type (
	// ComponentFunc is a function that takes a context and props and returns a Component.
	ComponentFunc func(context.Context) (Component, error)

	// Component represents a Construct. Constructs are the basic building block of cdk8s.
	// They are the instrument that enables composition and creation of higher-level abstractions through normal
	// object-oriented classes.
	Component interface {
		port.Component
	}
)

// ResourceID generates a resource id for the given prefix and props.
func ResourceID(prefix string, props Props) *string {
	return infra.ResourceID(prefix, props)
}

// ExtractResource extracts the resource name from the given generated resource id.
// This method is exposed primarily for testing purposes.
func ExtractResource(id *string) string {
	return infra.ExtractResource(id)
}
