package loki

import (
	"testing"

	"github.com/gkampitakis/go-snaps/snaps"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/crib-sdk/crib/scalar/helmchart/v1"
	"github.com/smartcontractkit/crib-sdk/internal"
)

func TestNewLokiChart(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test in short mode.")
	}
	t.Parallel()
	internal.JSIIKernelMutex.Lock()
	defer internal.JSIIKernelMutex.Unlock()
	is := assert.New(t)

	ctx := t.Context()
	l, err := internal.NewHelmValuesLoader(ctx, "testdata")
	require.NoError(t, err)
	userValues, err := l.Values()
	require.NoError(t, err)

	app := internal.NewTestApp(t)
	ctx = internal.ContextWithConstruct(ctx, app.Chart)
	lokiProps := &helmchart.ChartProps{
		Namespace: "ns-loki",
		Values:    userValues,
	}

	component, err := New(ctx, lokiProps)
	is.NoError(err)
	is.NotNil(component)

	snaps.MatchStandaloneYAML(t, *app.SynthYaml())
}
