package namespacev1

import (
	"testing"

	"github.com/gkampitakis/go-snaps/snaps"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/crib-sdk/crib"
	"github.com/smartcontractkit/crib-sdk/internal"
	"github.com/smartcontractkit/crib-sdk/internal/core/domain"
)

func TestValidate(t *testing.T) {
	t.Parallel()

	p := Props{Namespace: "foo-bar-baz"}
	assert.NoError(t, p.Validate(t.Context()))
}

func TestComponent(t *testing.T) {
	t.Parallel()
	is := assert.New(t)

	internal.JSIIKernelMutex.Lock()
	t.Cleanup(internal.JSIIKernelMutex.Unlock)

	app := internal.NewTestApp(t)
	ctx := internal.ContextWithConstruct(t.Context(), app.Chart)

	cf := Component("foo-bar-baz")
	is.NotNil(cf, "expected component to be non-nil")

	c, err := cf(ctx)
	require.NoError(t, err, "expected no error when creating component")
	is.Implements((*crib.Component)(nil), c, "expected component to implement crib.Component interface")

	raw := *app.SynthYaml()
	is.Contains(raw, "foo-bar-baz", "expected synthesized YAML to contain the namespace")
}

func TestNewNamespace(t *testing.T) {
	t.Parallel()
	internal.JSIIKernelMutex.Lock()
	t.Cleanup(internal.JSIIKernelMutex.Unlock)
	is := assert.New(t)

	app := internal.NewTestApp(t)
	ctx := internal.ContextWithConstruct(t.Context(), app.Chart)

	testProps := &Props{
		Namespace: "foo-bar-baz",
	}

	component, err := New(ctx, testProps)
	is.NoError(err)
	is.NotNil(component)

	raw := *app.SynthYaml()
	for manifest, err := range domain.UnmarshalDocument([]byte(raw)) {
		if err != nil {
			assert.NoError(t, err, "failed to unmarshal manifest: %s", manifest)
		}
		snaps.MatchStandaloneYAML(t, manifest)
	}
}
