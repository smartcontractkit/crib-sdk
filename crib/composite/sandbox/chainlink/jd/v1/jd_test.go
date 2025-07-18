package jdv1

import (
	"testing"

	"github.com/cdk8s-team/cdk8s-core-go/cdk8s/v2"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"

	"github.com/smartcontractkit/crib-sdk/crib"
	"github.com/smartcontractkit/crib-sdk/internal"
	"github.com/smartcontractkit/crib-sdk/internal/core/common/dry"
)

// TestComponent verifies that the JD component is properly constructed
// with all required charts and configurations.
func TestComponent(t *testing.T) {
	t.Parallel()
	internal.JSIIKernelMutex.Lock()
	t.Cleanup(internal.JSIIKernelMutex.Unlock)
	is := assert.New(t)

	// Setup test environment
	app := internal.NewTestApp(t)
	ctx := internal.ContextWithConstruct(t.Context(), app.Chart)

	// Define test properties
	testProps := &Props{
		DB: DBProps{
			Namespace:   "test-namespace",
			ReleaseName: "test-jd-db",
			Values: map[string]any{
				"key": "value",
			},
			ValuesPatches: [][]string{
				{"path", "to", "value"},
			},
		},
		JD: JDProps{
			Namespace:   "test-namespace",
			ReleaseName: "test-jd",
			Values: map[string]any{
				"test": "value",
			},
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
		"sdk.HelmChart#postgres",
		"sdk.HelmChart#jd",
		"sdk.Namespace",
		"sdk.Namespace",
		"sdk.HelmChart",
		"sdk.ClientSideApply",
		"sdk.HelmChart",
		"sdk.ClientSideApply",
	}
	is.Equal(wantCharts, gotCharts, "Expected charts should match actual charts")
	is.Len(*app.Charts(), len(wantCharts), "Number of charts should match expected count")

	// Find specific charts for testing
	var postgres, jd, waitFor cdk8s.Chart
	for _, c := range *app.Charts() {
		if !dry.FromPtr(cdk8s.Chart_IsChart(c)) {
			continue
		}

		switch crib.ExtractResource(c.Node().Id()) {
		case "sdk.HelmChart#postgres":
			postgres = c
		case "sdk.HelmChart#jd":
			jd = c
		case "sdk.ClientSideApply":
			// Get the last ClientSideApply (JD wait)
			waitFor = c
		}
	}

	// Verify required charts exist
	is.NotNil(postgres, "Postgres chart should be present")
	is.NotNil(jd, "JD chart should be present")
	is.NotNil(waitFor, "WaitFor chart should be present")

	// Test ClientSideApply configuration
	t.Run("ClientSideApply", func(t *testing.T) {
		var obj cdk8s.ApiObject
		is.NotPanics(func() {
			obj = cdk8s.ApiObject_Of(waitFor.Node().DefaultChild())
		}, "Should not panic when getting default child")
		is.NotNil(obj, "ClientSideApply object should not be nil")

		// Verify object metadata
		is.Equal("crib.smartcontract.com/v1alpha1", *obj.ApiVersion(), "API version should match")
		is.Equal("ClientSideApply", *obj.Kind(), "Kind should be ClientSideApply")
		is.Equal("crib.smartcontract.com", *obj.ApiGroup(), "API group should match")
		is.Equal("test-namespace", *obj.Metadata().Namespace(), "Namespace should match")

		// Verify object specification
		json := dry.As[map[string]any](obj.ToJson())
		is.NotNil(json, "JSON representation should not be nil")
		spec := dry.As[map[string]any](json["spec"])
		is.NotNil(spec, "Spec should not be nil")

		want := map[string]any{
			"onFailure": "abort",
			"action":    "kubectl",
			"args": []any{
				"wait",
				"-n", "test-namespace",
				"--for=condition=ready",
				"pod",
				"-l=app.kubernetes.io/component=test-jd,app.kubernetes.io/name=devspace-app",
				"--timeout=600s",
			},
		}
		is.Equal(want, spec, "Spec should match expected configuration")
	})
}
