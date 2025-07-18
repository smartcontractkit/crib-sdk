package helm

import (
	"errors"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/crib-sdk/internal/adapter/filehandler"
	"github.com/smartcontractkit/crib-sdk/internal/core/domain"
)

func TestValidate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		chart *Chart
		want  assert.ErrorAssertionFunc
	}{
		{
			name: "valid application chart",
			chart: &Chart{
				Type: domain.HelmChartTypeApplication,
			},
			want: assert.NoError,
		},
		{
			name: "valid library chart",
			chart: &Chart{
				Type: domain.HelmChartTypeLibrary,
			},
			want: assert.NoError,
		},
		{
			name:  "valid chart with empty type",
			chart: &Chart{},
			want:  assert.NoError,
		},
		{
			name: "invalid chart with unsupported type",
			chart: &Chart{
				Type: "unsupported",
			},
			want: assert.Error,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			tc.want(t, tc.chart.Validate(t.Context()))
		})
	}
}

func TestUnmarshal(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected Chart
		wantErr  assert.ErrorAssertionFunc
	}{
		{
			name:    "valid chart - missing type",
			input:   "Chart1.yaml",
			wantErr: assert.NoError,
			expected: Chart{
				Type: domain.HelmChartTypeApplication, // Default type if not specified
			},
		},
		{
			name:  "valid library chart",
			input: "Chart2.yaml",
			expected: Chart{
				Type: domain.HelmChartTypeLibrary,
			},
			wantErr: assert.NoError,
		},
		{
			name:  "valid application chart",
			input: "Chart3.yaml",
			expected: Chart{
				Type: domain.HelmChartTypeApplication,
			},
			wantErr: assert.NoError,
		},
		{
			name:    "invalid chart type",
			input:   "Chart4.yaml",
			wantErr: assert.Error,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()

			// Create a temporary file handler for the test.
			tmp, err := filehandler.NewTempHandler(ctx, tc.name)
			require.NoError(t, err)
			// Create the reader for the Chart.
			testdata, err := filehandler.New(ctx, "testdata")
			require.NoError(t, err)

			// Copy the test data into the temporary file.
			testChart, err := testdata.Open(tc.input)
			require.NoError(t, err)
			newChart, err := tmp.Create(domain.HelmChartFileName)
			require.NoError(t, err)
			_, err = io.Copy(newChart, testChart)
			require.NoError(t, func() error {
				return errors.Join(err, newChart.Close(), testChart.Close())
			}())

			var chart Chart
			gotErr := chart.Unmarshal(ctx, tmp)
			if tc.wantErr(t, gotErr) && gotErr == nil {
				assert.Equal(t, tc.expected, chart)
			}
		})
	}
}
