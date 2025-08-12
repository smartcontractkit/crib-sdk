package internal

import (
	"context"
	"testing"

	"github.com/cdk8s-team/cdk8s-core-go/cdk8s/v2"
	"github.com/gkampitakis/go-snaps/snaps"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/crib-sdk/internal/core/common/dry"
	"github.com/smartcontractkit/crib-sdk/internal/core/common/infra"
	"github.com/smartcontractkit/crib-sdk/internal/core/domain"
	"github.com/smartcontractkit/crib-sdk/internal/core/service/iresolver"
)

const testAppName = "TestingApp"

// TestApp exposes the cdk8s.App and cdk8s.Chart types for use in unit tests.
type TestApp struct {
	cdk8s.App
	cdk8s.Chart

	t     *testing.T
	ctxFn func() context.Context
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
	chart := cdk8s.NewChart(app, infra.ResourceID(testAppName, nil), nil)

	ctx := t.Context()
	ctx = ContextWithConstruct(ctx, chart)

	return &TestApp{
		App:   app,
		Chart: chart,
		t:     t,
		ctxFn: func() context.Context { return ctx },
	}
}

// Context returns the context for the TestApp, which is used to pass around
// the constructs and other values.
func (app *TestApp) Context() context.Context {
	if app.ctxFn == nil {
		return context.TODO()
	}
	return app.ctxFn()
}

// SynthYaml calls the SynthAndSnapYamls method to synthesize the YAML output and
// write the snapshots into separate files.
//
//nolint:revive // SynthYaml matches the cdk8s method name.
func (app *TestApp) SynthYaml() *string {
	return SynthAndSnapYamls(app.t, app)
}

// SynthAndSnapYamls calls SynthYaml and unmarshal results into separate files.
func SynthAndSnapYamls(t *testing.T, app *TestApp) *string {
	require.NotNil(t, app, "app must not be nil")
	raw := app.App.SynthYaml()
	for obj, err := range domain.UnmarshalDocument([]byte(dry.FromPtr(raw))) {
		if err != nil {
			assert.NoError(t, err)
		}
		snaps.MatchStandaloneYAML(t, obj)
	}
	return raw
}
