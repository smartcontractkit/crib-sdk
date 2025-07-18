package internal

import (
	"testing"

	"github.com/cdk8s-team/cdk8s-core-go/cdk8s/v2"
	"github.com/gkampitakis/go-snaps/snaps"
	"github.com/stretchr/testify/assert"

	"github.com/smartcontractkit/crib-sdk/internal/core/common/dry"
	"github.com/smartcontractkit/crib-sdk/internal/core/common/infra"
	"github.com/smartcontractkit/crib-sdk/internal/core/domain"
	"github.com/smartcontractkit/crib-sdk/internal/core/service/iresolver"
)

// TestApp exposes the cdk8s.App and cdk8s.Chart types for use in unit tests.
type TestApp struct {
	cdk8s.App
	cdk8s.Chart
}

// NewTestApp creates a new test Chart scope for use in unit tests.
func NewTestApp(t *testing.T) *TestApp {
	t.Helper()

	// Check if we're in a test environment.
	if !testing.Testing() {
		panic("This method should only be used in test environments.")
	}

	// Create a new App instance.
	app := cdk8s.Testing_App(&cdk8s.AppProps{
		YamlOutputType: cdk8s.YamlOutputType_FILE_PER_APP,
		Resolvers: dry.ToPtr([]cdk8s.IResolver{
			iresolver.NewResolver(iresolver.NameResolver, iresolver.ResolutionPriorityLow),
		}),
	})
	// Create a new Chart scope and return.
	chart := cdk8s.NewChart(app, infra.ResourceID("TestingApp", nil), &cdk8s.ChartProps{})
	return &TestApp{
		App:   app,
		Chart: chart,
	}
}

// SynthAndSnapYamls calls SynthYaml and unmarshall results into separate files.
func SynthAndSnapYamls(t *testing.T, app *TestApp) {
	raw := *app.SynthYaml()
	for obj, err := range domain.UnmarshalDocument([]byte(raw)) {
		if err != nil {
			assert.NoError(t, err)
		}
		snaps.MatchStandaloneYAML(t, obj)
	}
}
