package jdv1

import (
	"testing"

	"github.com/cdk8s-team/cdk8s-core-go/cdk8s/v2"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"

	"github.com/smartcontractkit/crib-sdk/crib"
	"github.com/smartcontractkit/crib-sdk/internal"
)

// TestComponent_MinimalProps verifies that the JD component works with minimal required properties.
func TestComponent_MinimalProps(t *testing.T) {
	t.Parallel()
	internal.JSIIKernelMutex.Lock()
	t.Cleanup(internal.JSIIKernelMutex.Unlock)
	is := assert.New(t)

	// Setup test environment
	app := internal.NewTestApp(t)
	ctx := internal.ContextWithConstruct(t.Context(), app.Chart)

	// Define minimal test properties
	testProps := &Props{
		AppInstanceName: "test-jd",
		Namespace:       "test-namespace",
		DB:              DBProps{},
		JD: JDProps{
			Image:            "test-jd:1.4.3",
			CSAEncryptionKey: "d1093c0060d50a3c89c189b2e485da5a3ce57f3dcb38ab7e2c0d5f0bb2314a44",
		},
	}

	// Create and validate component
	c := Component(testProps)
	component, err := c(ctx)
	is.NoError(err, "Component creation should not return an error")
	is.NotNil(component, "Component should not be nil")

	// Verify chart structure
	gotCharts := lo.Map(*app.Charts(), func(c cdk8s.Chart, _ int) string {
		return crib.ExtractResource(c.Node().Id())
	})
	wantCharts := []string{
		"TestingApp",
		ComponentName,
		"sdk.HelmChart#postgres", "sdk.Namespace", "sdk.HelmChart", "sdk.ClientSideApply", "sdk.DeploymentV1", "sdk.ServiceV1",
	}
	is.Equal(wantCharts, gotCharts, "Expected charts should match actual charts")
	is.Len(*app.Charts(), len(wantCharts), "Number of charts should match expected count")

	internal.SynthAndSnapYamls(t, app)
}

// TestComponent_AllProps verifies that the JD component works with all properties including optional ones.
func TestComponent_AllProps(t *testing.T) {
	t.Parallel()
	internal.JSIIKernelMutex.Lock()
	t.Cleanup(internal.JSIIKernelMutex.Unlock)
	is := assert.New(t)

	// Setup test environment
	app := internal.NewTestApp(t)
	ctx := internal.ContextWithConstruct(t.Context(), app.Chart)

	// Define test properties with all fields populated
	testProps := &Props{
		AppInstanceName: "test-jd",
		Namespace:       "test-namespace",
		WaitForRollout:  true,
		DB: DBProps{
			Values: map[string]any{
				"postgresql": map[string]any{
					"auth": map[string]any{
						"postgresPassword": "testpassword",
					},
				},
			},
			ValuesPatches: [][]string{
				{"postgresql.primary.persistence.size", "10Gi"},
				{"postgresql.auth.database", "jd_test"},
			},
		},
		JD: JDProps{
			Image:            "test-jd:1.4.3",
			CSAEncryptionKey: "d1093c0060d50a3c89c189b2e485da5a3ce57f3dcb38ab7e2c0d5f0bb2314a44",
		},
	}

	// Create and validate component
	c := Component(testProps)
	component, err := c(ctx)
	is.NoError(err, "Component creation should not return an error")
	is.NotNil(component, "Component should not be nil")

	// Verify chart structure
	gotCharts := lo.Map(*app.Charts(), func(c cdk8s.Chart, _ int) string {
		return crib.ExtractResource(c.Node().Id())
	})
	wantCharts := []string{
		"TestingApp",
		ComponentName,
		"sdk.HelmChart#postgres", "sdk.Namespace", "sdk.HelmChart", "sdk.ClientSideApply", "sdk.ServiceV1", "sdk.DeploymentV1", "sdk.ClientSideApply",
	}
	is.Equal(wantCharts, gotCharts, "Expected charts should match actual charts")
	is.Len(*app.Charts(), len(wantCharts), "Number of charts should match expected count")

	internal.SynthAndSnapYamls(t, app)
}

// TestComponent_ValidationFailure verifies that the component fails with invalid properties.
func TestComponent_ValidationFailure(t *testing.T) {
	t.Parallel()
	internal.JSIIKernelMutex.Lock()
	t.Cleanup(internal.JSIIKernelMutex.Unlock)
	is := assert.New(t)

	// Setup test environment
	app := internal.NewTestApp(t)
	ctx := internal.ContextWithConstruct(t.Context(), app.Chart)

	testCases := []struct {
		name  string
		props *Props
	}{
		{
			name: "missing Namespace",
			props: &Props{
				DB: DBProps{},
				JD: JDProps{
					Image:            "test-jd:1.4.3",
					CSAEncryptionKey: "d1093c0060d50a3c89c189b2e485da5a3ce57f3dcb38ab7e2c0d5f0bb2314a44",
				},
			},
		},
		{
			name: "invalid CSA encryption key length",
			props: &Props{
				Namespace: "test-namespace",
				DB:        DBProps{},
				JD: JDProps{
					Image:            "test-jd:1.4.3",
					CSAEncryptionKey: "invalidkey",
				},
			},
		},
		{
			name: "invalid CSA encryption key format",
			props: &Props{
				Namespace: "test-namespace",
				DB:        DBProps{},
				JD: JDProps{
					Image:            "test-jd:1.4.3",
					CSAEncryptionKey: "gggggggggggggggggggggggggggggggggggggggggggggggggggggggggggggggg",
				},
			},
		},
		{
			name: "invalid image URI",
			props: &Props{
				Namespace: "test-namespace",
				DB:        DBProps{},
				JD: JDProps{
					Image:            "invalid image uri with spaces",
					CSAEncryptionKey: "d1093c0060d50a3c89c189b2e485da5a3ce57f3dcb38ab7e2c0d5f0bb2314a44",
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			c := Component(tc.props)
			component, err := c(ctx)
			is.Error(err, "Component creation should return an error for invalid props")
			is.Nil(component, "Component should be nil when validation fails")
		})
	}
}
