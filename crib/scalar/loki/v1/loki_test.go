package loki

import (
	"testing"

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
	is := assert.New(t)
	app := internal.NewTestApp(t)

	l, err := internal.NewHelmValuesLoader(app.Context(), "testdata")
	require.NoError(t, err)
	userValues, err := l.Values()
	require.NoError(t, err)

	lokiProps := &helmchart.ChartProps{
		Namespace: "ns-loki",
		Values:    userValues,
	}

	component, err := New(app.Context(), lokiProps)
	is.NoError(err)
	is.NotNil(component)

	app.SynthYaml()
}
