package internal

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChartLoader(t *testing.T) {
	t.Parallel()
	fs, err := os.OpenRoot("testdata")
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, fs.Close())
	})

	is := assert.New(t)
	ref, err := NewChartRef(fs.FS(), "chart.defaults.yaml")
	is.NoError(err)
	is.NotNil(ref)

	want := &ChartRef{
		Chart: Chart{
			Name:        "test-chart",
			ReleaseName: "test-release",
			Repository:  "https://helm.example.com/charts",
			Version:     "1.0.0",
		},
		Values: map[string]any{
			"one":   "two",
			"two":   1,
			"three": true,
			"four":  []any{"one", "two"},
			"five": map[string]any{
				"one": "two",
				"two": 1,
			},
		},
	}
	is.Equal(want, ref)
}
