package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/crib-sdk/internal/adapter/filehandler"
)

func Test_findManifests(t *testing.T) {
	t.Parallel()

	fh := setup(t, "testdata/plan/manifests/basic")
	ps := &PlanService{
		fh: fh,
	}
	manifests := ps.findManifests()
	want := map[string][]Manifest{
		"00": {
			{Name: "00/00-a.yaml"},
			{Name: "00/01-b.yaml"},
		},
		"01": {
			{Name: "01/00-c.yaml"},
			{Name: "01/01-d.yaml", IsLocal: true},
			{Name: "01/02-e.yaml"},
		},
		"02-client_side_apply": {
			{Name: "02-client_side_apply/00-cmd.yaml", IsLocal: true},
			{Name: "02-client_side_apply/01-task.yaml", IsLocal: true},
			{Name: "02-client_side_apply/02-cribctl.yaml", IsLocal: true},
			{Name: "02-client_side_apply/03-kubectl.yaml", IsLocal: true},
		},
	}
	assert.Equal(t, manifests, want)
}

func Test_isLocalManifest(t *testing.T) {
	t.Parallel()

	fh := setup(t, "testdata/plan/manifests/basic")
	ps := &PlanService{
		fh: fh,
	}

	tests := []struct {
		name     string
		filename string
		expected assert.BoolAssertionFunc
	}{
		{
			name:     "local manifest",
			filename: "01/01-d.yaml",
			expected: assert.True,
		},
		{
			name:     "remote manifest",
			filename: "00/01-b.yaml",
			expected: assert.False,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.expected(t, ps.isLocalManifest(tc.filename))
		})
	}
}

func Test_discoverYAML(t *testing.T) {
	t.Parallel()

	fh := setup(t, "testdata/plan/manifests/basic")
	ps := &PlanService{
		fh: fh,
	}

	tests := []struct {
		name     string
		filename string
		expected assert.ErrorAssertionFunc
	}{
		{
			name:     "yaml file",
			filename: "00/00-a.yaml",
			expected: assert.NoError,
		},
		{
			name:     "non-yaml file",
			filename: "00/README.md",
			expected: assert.Error,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			tc.expected(t, ps.discoverYAML(nil, tc.filename))
		})
	}
}

func Test_normalizeManifests(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		manifests map[string][]Manifest
		expected  []ManifestBundle
	}{
		{
			name: "one dir, multiple local",
			manifests: map[string][]Manifest{
				"00": {
					{Name: "00/00-a.yaml", IsLocal: true},
					{Name: "00/01-b.yaml", IsLocal: true},
					{Name: "00/02-c.yaml", IsLocal: true},
				},
			},
			expected: []ManifestBundle{
				{
					isLocal:   true,
					manifests: []Manifest{{Name: "00/00-a.yaml", IsLocal: true}},
				},
				{
					isLocal:   true,
					manifests: []Manifest{{Name: "00/01-b.yaml", IsLocal: true}},
				},
				{
					isLocal:   true,
					manifests: []Manifest{{Name: "00/02-c.yaml", IsLocal: true}},
				},
			},
		},
		{
			name: "two dirs, one local",
			manifests: map[string][]Manifest{
				"00": {
					{Name: "00/00-a.yaml"},
					{Name: "00/01-b.yaml"},
					{Name: "00/02-c.yaml", IsLocal: true},
				},
				"01": {
					{Name: "01/00-d.yaml"},
					{Name: "01/01-e.yaml"},
				},
			},
			expected: []ManifestBundle{
				{
					manifests: []Manifest{
						{Name: "00/00-a.yaml"},
						{Name: "00/01-b.yaml"},
					},
				},
				{
					isLocal: true,
					manifests: []Manifest{
						{Name: "00/02-c.yaml", IsLocal: true},
					},
				},
				{
					manifests: []Manifest{
						{Name: "01/00-d.yaml"},
						{Name: "01/01-e.yaml"},
					},
				},
			},
		},
		{
			name: "single dir, all local",
			manifests: map[string][]Manifest{
				"00": {
					{Name: "00/00-a.yaml", IsLocal: true},
					{Name: "00/01-b.yaml", IsLocal: true},
					{Name: "00/02-c.yaml", IsLocal: true},
				},
			},
			expected: []ManifestBundle{
				{
					isLocal:   true,
					manifests: []Manifest{{Name: "00/00-a.yaml", IsLocal: true}},
				},
				{
					isLocal:   true,
					manifests: []Manifest{{Name: "00/01-b.yaml", IsLocal: true}},
				},
				{
					isLocal:   true,
					manifests: []Manifest{{Name: "00/02-c.yaml", IsLocal: true}},
				},
			},
		},
		{
			name: "single dir, no local",
			manifests: map[string][]Manifest{
				"00": {
					{Name: "00/00-a.yaml"},
					{Name: "00/01-b.yaml"},
				},
			},
			expected: []ManifestBundle{
				{
					manifests: []Manifest{
						{Name: "00/00-a.yaml"},
						{Name: "00/01-b.yaml"},
					},
				},
			},
		},
		{
			name: "two dirs, no local",
			manifests: map[string][]Manifest{
				"00": {
					{Name: "00/00-a.yaml"},
					{Name: "00/01-b.yaml"},
				},
				"01": {
					{Name: "01/00-c.yaml"},
					{Name: "01/01-d.yaml"},
				},
			},
			expected: []ManifestBundle{
				{
					manifests: []Manifest{
						{Name: "00/00-a.yaml"},
						{Name: "00/01-b.yaml"},
					},
				},
				{
					manifests: []Manifest{
						{Name: "01/00-c.yaml"},
						{Name: "01/01-d.yaml"},
					},
				},
			},
		},
		{
			name: "real-world example",
			manifests: map[string][]Manifest{
				"00": {
					{Name: "00/00-a.yaml"},
					{Name: "00/01-b.yaml"},
				},
				"01": {
					{Name: "01/00-c.yaml"},
					{Name: "01/01-d.yaml", IsLocal: true},
					{Name: "01/02-e.yaml"},
				},
				"02-client_side_apply": {
					{Name: "02-client_side_apply/00-cmd.yaml", IsLocal: true},
					{Name: "02-client_side_apply/01-task.yaml", IsLocal: true},
					{Name: "02-client_side_apply/02-cribctl.yaml", IsLocal: true},
					{Name: "02-client_side_apply/03-kubectl.yaml", IsLocal: true},
				},
			},
			expected: []ManifestBundle{
				{
					manifests: []Manifest{
						{Name: "00/00-a.yaml"},
						{Name: "00/01-b.yaml"},
					},
				},
				{
					manifests: []Manifest{
						{Name: "01/00-c.yaml"},
					},
				},
				{
					isLocal: true,
					manifests: []Manifest{
						{Name: "01/01-d.yaml", IsLocal: true},
					},
				},
				{
					manifests: []Manifest{
						{Name: "01/02-e.yaml"},
					},
				},
				{
					isLocal: true,
					manifests: []Manifest{
						{Name: "02-client_side_apply/00-cmd.yaml", IsLocal: true},
					},
				},
				{
					isLocal: true,
					manifests: []Manifest{
						{Name: "02-client_side_apply/01-task.yaml", IsLocal: true},
					},
				},
				{
					isLocal: true,
					manifests: []Manifest{
						{Name: "02-client_side_apply/02-cribctl.yaml", IsLocal: true},
					},
				},
				{
					isLocal: true,
					manifests: []Manifest{
						{Name: "02-client_side_apply/03-kubectl.yaml", IsLocal: true},
					},
				},
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			fh, err := filehandler.NewTempHandler(t.Context(), tc.name)
			require.NoError(t, err)
			t.Cleanup(func() {
				require.NoError(t, fh.RemoveAll())
			})

			p := &PlanService{fh: fh}
			result := p.normalizeManifests(tc.manifests)
			// Provide the root to the ManifestBundle for comparison.
			for i := range tc.expected {
				tc.expected[i].root = fh.Name()
			}
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestApplyPlan(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	fh := setup(t, "testdata/plan/manifests/basic")
	ps := &PlanService{
		fh: fh,
	}
	plan := &AppPlan{
		svc: ps,
	}

	// Apply the plan.
	res, err := plan.Apply(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, res)
}

func TestApply(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	fh := setup(t, "testdata/plan/manifests/basic")
	ps := &PlanService{
		fh: fh,
	}

	tests := []struct {
		name     string
		bundle   ManifestBundle
		expected assert.ErrorAssertionFunc
	}{
		{
			name: "remote",
			bundle: ManifestBundle{
				manifests: []Manifest{
					{Name: "00/00-a.yaml"},
					{Name: "00/01-b.yaml"},
					{Name: "00/02 with spaces.yaml"},
				},
			},
			expected: assert.NoError,
		},
		{
			name: "local - cmd",
			bundle: ManifestBundle{
				isLocal: true,
				manifests: []Manifest{
					{Name: "02-client_side_apply/00-cmd.yaml", IsLocal: true},
				},
			},
			expected: assert.NoError,
		},
		{
			name: "local - task",
			bundle: ManifestBundle{
				isLocal: true,
				manifests: []Manifest{
					{Name: "02-client_side_apply/01-task.yaml", IsLocal: true},
				},
			},
			expected: assert.NoError,
		},
		{
			name: "local - cribctl",
			bundle: ManifestBundle{
				isLocal: true,
				manifests: []Manifest{
					{Name: "02-client_side_apply/02-cribctl.yaml", IsLocal: true},
				},
			},
			expected: assert.NoError,
		},
		{
			name: "local - kubectl",
			bundle: ManifestBundle{
				isLocal: true,
				manifests: []Manifest{
					{Name: "02-client_side_apply/03-kubectl.yaml", IsLocal: true},
				},
			},
			expected: assert.NoError,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := tc.bundle.Apply(ctx, ps)
			tc.expected(t, err)
		})
	}
}

func TestManifestBundleString(t *testing.T) {
	t.Parallel()

	mb := ManifestBundle{
		manifests: []Manifest{
			{Name: "00-namespace.yaml"},
			{Name: "01-deployment.yaml"},
			{Name: "02 with spaces.yaml"},
		},
	}

	expected := "00-namespace.yaml,01-deployment.yaml,02 with spaces.yaml"
	assert.Equal(t, expected, mb.String())
}

// setup is a helper that copies the test data to a temporary directory.
func setup(t *testing.T, basePath string) *filehandler.Handler {
	t.Helper()
	must := require.New(t)

	// Copy the test data to a temporary directory.
	src, err := filehandler.New(t.Context(), basePath)
	must.NoError(err)
	dst, err := filehandler.New(t.Context(), t.TempDir())
	must.NoError(err)

	must.NoError(filehandler.CopyDir(src, dst))
	return dst
}
