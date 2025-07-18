package helm

import (
	"testing"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/gkampitakis/go-snaps/snaps"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/crib-sdk/internal/adapter/filehandler"
	"github.com/smartcontractkit/crib-sdk/internal/core/domain"
	"github.com/smartcontractkit/crib-sdk/internal/core/port"
)

func TestRelease(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		in      port.ChartReleaser
		IsOCI   assert.BoolAssertionFunc
		PullRef assert.ComparisonAssertionFunc
	}{
		{
			name: "https repo",
			in: &Release{
				Name:        "test-chart",
				ReleaseName: "test-release",
				Repository:  "https://example.com/charts",
				Version:     "1.0.0",
			},
			IsOCI:   assert.False,
			PullRef: assert.Equal,
		},
		{
			name: "oci repo",
			in: &Release{
				Name:        "test-chart",
				ReleaseName: "test-release",
				Repository:  "oci://example.com/charts",
				Version:     "1.0.0",
			},
			IsOCI:   assert.True,
			PullRef: assert.NotEqual,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			tc.IsOCI(t, tc.in.IsOCI())
			tc.PullRef(t, tc.in.PullRef(), tc.in.String())
		})
	}
}

func TestDefaultsSave(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		defaults *Defaults
		wantErr  assert.ErrorAssertionFunc
		wantFile assert.BoolAssertionFunc
	}{
		{
			name: "valid defaults - v prefix missing on version",
			defaults: &Defaults{
				Release: Release{
					Name:        "test-chart",
					ReleaseName: "test-release",
					Repository:  "https://example.com/charts",
					Version:     "0.9.1", // Version without 'v' prefix
				},
				Values: map[string]any{
					"key1": "value1",
					"key2": 42,
					"key3": true,
					"obj": map[string]any{
						"nestedKey1": "nestedValue1",
					},
				},
			},
			wantErr:  assert.NoError,
			wantFile: assert.True,
		},
		{
			name: "valid defaults - v prefix on version",
			defaults: &Defaults{
				Release: Release{
					Name:        "test-chart",
					ReleaseName: "test-release",
					Repository:  "https://example.com/charts",
					Version:     "v0.9.1",
				},
				Values: map[string]any{
					"key1": "value1",
					"key2": 42,
					"key3": true,
					"obj": map[string]any{
						"nestedKey1": "nestedValue1",
					},
				},
			},
			wantErr:  assert.NoError,
			wantFile: assert.True,
		},
		{
			name: "valid defaults - empty values",
			defaults: &Defaults{
				Release: Release{
					Name:        "test-chart",
					ReleaseName: "test-release",
					Repository:  "https://example.com/charts",
					Version:     "1.0.0",
				},
			},
			wantErr:  assert.NoError,
			wantFile: assert.True,
		},
		{
			name: "invalid defaults - invalid version",
			defaults: &Defaults{
				Release: Release{
					Name:        "test-chart",
					ReleaseName: "test-release",
					Repository:  "https://example.com/charts",
					Version:     gofakeit.Word(), // Invalid version
				},
			},
			wantErr:  assert.Error,
			wantFile: assert.False,
		},
		{
			name: "invalid defaults - missing name",
			defaults: &Defaults{
				Release: Release{
					ReleaseName: "test-release",
					Repository:  "https://example.com/charts",
					Version:     "1.0.0",
				},
			},
			wantErr:  assert.Error,
			wantFile: assert.False,
		},
		{
			name: "invalid defaults - missing repository",
			defaults: &Defaults{
				Release: Release{
					Name:        "test-chart",
					ReleaseName: "test-release",
					Version:     "1.0.0",
				},
			},
			wantErr:  assert.Error,
			wantFile: assert.False,
		},
		{
			name: "invalid defaults - missing version",
			defaults: &Defaults{
				Release: Release{
					Name:        "test-chart",
					ReleaseName: "test-release",
					Repository:  "https://example.com/charts",
				},
			},
			wantErr:  assert.Error,
			wantFile: assert.False,
		},
		{
			name: "invalid defaults - empty release",
			defaults: &Defaults{
				Release: Release{},
			},
			wantErr:  assert.Error,
			wantFile: assert.False,
		},
		{
			name:     "invalid defaults - missing release",
			defaults: &Defaults{},
			wantErr:  assert.Error,
			wantFile: assert.False,
		},
		{
			name: "invalid defaults - invalid repository",
			defaults: &Defaults{
				Release: Release{
					Name:        "test-chart",
					ReleaseName: "test-release",
					Repository:  "git://example.com/charts", // Invalid repository URL
					Version:     "1.0.0",
				},
			},
			wantErr:  assert.Error,
			wantFile: assert.False,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()

			fh, err := filehandler.NewTempHandler(ctx, tc.name)
			require.NoError(t, err)

			saveErr := tc.defaults.Save(ctx, fh)
			tc.wantErr(t, saveErr)
			tc.wantFile(t, fh.FileExists(domain.HelmDefaultsFileName))
			if saveErr != nil {
				return // Skip further checks if saving failed.
			}

			var newDefaults Defaults
			err = newDefaults.Unmarshal(ctx, fh)
			require.NotNil(t, newDefaults)
			require.NoError(t, err)

			snaps.MatchStandaloneYAML(t, newDefaults)
		})
	}
}
