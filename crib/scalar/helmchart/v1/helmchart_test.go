package helmchart

import (
	"bytes"
	"testing"

	"github.com/gkampitakis/go-snaps/match"
	"github.com/gkampitakis/go-snaps/snaps"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/smartcontractkit/crib-sdk/internal"
)

func TestNewHTTPSHelmChart(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test in short mode.")
	}
	t.Parallel()

	internal.JSIIKernelMutex.Lock()
	t.Cleanup(internal.JSIIKernelMutex.Unlock)

	is := assert.New(t)

	l, err := internal.NewHelmValuesLoader(t.Context(), "testdata")
	require.NoError(t, err)
	values, err := l.Values()
	require.NoError(t, err)

	app := internal.NewTestApp(t)
	ctx := internal.ContextWithConstruct(t.Context(), app.Chart)

	testProps := &ChartProps{
		Name:        "test-chart",
		Chart:       "component-chart",
		Namespace:   "ns-helm-chart",
		ReleaseName: "my-test-chart",
		Repo:        "https://charts.loft.sh",
		Values:      values,
		Version:     "0.9.1",
	}

	component, err := New(ctx, testProps)
	is.NoError(err)
	is.NotNil(component)

	internal.SynthAndSnapYamls(t, app)
}

func TestNewHTTPSHelmChartWithWait(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test in short mode.")
	}
	t.Parallel()

	internal.JSIIKernelMutex.Lock()
	t.Cleanup(internal.JSIIKernelMutex.Unlock)

	is := assert.New(t)

	l, err := internal.NewHelmValuesLoader(t.Context(), "testdata")
	require.NoError(t, err)
	values, err := l.Values()
	require.NoError(t, err)

	app := internal.NewTestApp(t)
	ctx := internal.ContextWithConstruct(t.Context(), app.Chart)

	testProps := &ChartProps{
		Name:         "test-chart",
		Chart:        "component-chart",
		Namespace:    "ns-helm-chart",
		ReleaseName:  "my-test-chart",
		Repo:         "https://charts.loft.sh",
		Values:       values,
		Version:      "0.9.1",
		WaitForReady: true,
	}

	component, err := New(ctx, testProps)
	is.NoError(err)
	is.NotNil(component)

	for _, chart := range *app.Charts() {
		t.Logf("Chart: %s", *chart.ToString())
	}

	internal.SynthAndSnapYamls(t, app)
}

func TestNewOCIHelmChart(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test in short mode.")
	}
	t.Parallel()

	internal.JSIIKernelMutex.Lock()
	t.Cleanup(internal.JSIIKernelMutex.Unlock)

	is := assert.New(t)

	l, err := internal.NewHelmValuesLoader(t.Context(), "testdata")
	require.NoError(t, err)
	values, err := l.Values()
	require.NoError(t, err)

	app := internal.NewTestApp(t)

	testProps := &ChartProps{
		Name:        "test-chart",
		Chart:       "postgresql",
		Namespace:   "ns-helm-chart",
		ReleaseName: "my-test-chart",
		Repo:        "oci://registry-1.docker.io/bitnamicharts/postgresql",
		Values:      values,
		Version:     "16.7.10",
	}

	component, err := New(app.Context(), testProps)
	is.NoError(err)
	is.NotNil(component)

	raw := *app.DisableSnapshots().SynthYaml()
	manifests := unmarshalManifests([]byte(raw))
	matchers := []match.YAMLMatcher{
		match.
			Any("$.data.postgres-password").
			ErrOnMissingPath(false),
	}
	for _, m := range manifests {
		snaps.MatchStandaloneYAML(t, m, matchers...)
	}
}

type genericManifest map[string]any

func unmarshalManifests(m []byte) []genericManifest {
	dec := yaml.NewDecoder(bytes.NewReader(m))

	var manifests []genericManifest
	for {
		var doc genericManifest
		if dec.Decode(&doc) != nil {
			break
		}
		manifests = append(manifests, doc)
	}

	return manifests
}
