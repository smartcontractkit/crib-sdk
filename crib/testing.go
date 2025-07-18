package crib

import (
	"context"
	"testing"

	"github.com/smartcontractkit/crib-sdk/internal"
)

// JSIIKernelMutex is global mutex to synchronize all parallel invocations of jsii kernel
// This is required to be used in Tests when test code contains any invocations to jsii kernel and uses t.Parallel()
// It can be also used in the productions code if it contains any parallel invocations.
type JSIIKernelMutex struct{}

// TestApp exposes the cdk8s.App and cdk8s.Chart types for use in unit tests.
type TestApp = internal.TestApp

func (m *JSIIKernelMutex) Lock() {
	internal.JSIIKernelMutex.Lock()
}

func (m *JSIIKernelMutex) Unlock() {
	internal.JSIIKernelMutex.Unlock()
}

// NewHelmValuesLoader is a helper function to create a loader capable of loading values from a Helm Chart values file.
func NewHelmValuesLoader(ctx context.Context, path string) (*internal.FileLoader, error) {
	return internal.NewHelmValuesLoader(ctx, path)
}

// NewTestApp creates a new test Chart scope for use in unit tests.
func NewTestApp(t *testing.T) *TestApp {
	return internal.NewTestApp(t)
}
