package tests

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/crib-sdk/internal"

	namespace "github.com/smartcontractkit/crib-sdk/crib/scalar/k8s/namespace/v1"
)

func TestMultipleNamespaces(t *testing.T) {
	t.Parallel()
	is := assert.New(t)

	goldenFile, err := os.ReadFile("testdata/multiple_namespaces.golden.yaml")
	require.NoError(t, err)

	app := internal.NewTestApp(t)
	ctx := internal.ContextWithConstruct(t.Context(), app.Chart)
	namespaces := []string{"foo", "bar", "baz"}
	for _, ns := range namespaces {
		t.Run(ns, func(t *testing.T) {
			// Note: Do not use t.Parallel() here. It will cause indeterministic output.

			props := &namespace.Props{Namespace: ns}
			component, err := namespace.New(ctx, props)
			is.NoError(err)
			is.NotNil(component)
		})
	}

	is.Equal(string(goldenFile), *app.SynthYaml())
}
