package crib

import (
	"context"
	"testing"

	"github.com/smartcontractkit/crib-sdk/internal"
)

// TestApp exposes the cdk8s.App and cdk8s.Chart types for use in unit tests.
type TestApp = internal.TestApp

// NewHelmValuesLoader is a helper function to create a loader capable of loading values from a Helm Chart values file.
func NewHelmValuesLoader(ctx context.Context, path string) (*internal.FileLoader, error) {
	return internal.NewHelmValuesLoader(ctx, path)
}

// NewTestApp creates a new test Chart scope for use in unit tests.
func NewTestApp(t *testing.T) *TestApp {
	return internal.NewTestApp(t)
}
