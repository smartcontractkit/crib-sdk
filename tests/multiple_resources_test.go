package tests

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/smartcontractkit/crib-sdk/internal"

	namespace "github.com/smartcontractkit/crib-sdk/crib/scalar/k8s/namespace/v1"
)

func TestMultipleNamespaces(t *testing.T) {
	t.Parallel()
	is := assert.New(t)

	app := internal.NewTestApp(t)
	namespaces := []string{"foo", "bar", "baz"}
	for _, ns := range namespaces {
		t.Run(ns, func(t *testing.T) {
			// Note: Do not use t.Parallel() here. It will cause indeterministic output.

			props := &namespace.Props{Namespace: ns}
			component, err := namespace.New(app.Context(), props)
			is.NoError(err)
			is.NotNil(component)
		})
	}

	app.SynthYaml()
}
