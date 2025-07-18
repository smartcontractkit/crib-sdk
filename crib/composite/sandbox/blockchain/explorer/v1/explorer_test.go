package blockchainexplorerv1

import (
	"testing"

	"github.com/cdk8s-team/cdk8s-core-go/cdk8s/v2"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"

	"github.com/smartcontractkit/crib-sdk/crib"
	"github.com/smartcontractkit/crib-sdk/internal"
	"github.com/smartcontractkit/crib-sdk/internal/core/common/dry"
	"github.com/smartcontractkit/crib-sdk/internal/core/domain"
)

func TestComponent(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test in short mode.")
	}
	t.Parallel()
	is := assert.New(t)

	app := internal.NewTestApp(t)
	ctx := internal.ContextWithConstruct(t.Context(), app.Chart)

	testProps := &Props{
		Namespace:   "test-namespace",
		ReleaseName: "test-blockexplorer",
		Type:        ExplorerTypeOtterscan,
		Values: map[string]any{
			"key": "value",
		},
		ValuesPatches: [][]string{
			{"path", "to", "value"},
		},
	}

	c := Component(testProps)
	component, err := c(ctx)
	is.NoError(err)
	is.NotNil(component)

	// Normalize the chart names into a more readable format.
	gotCharts := lo.Map(*app.Charts(), func(c cdk8s.Chart, _ int) string {
		return crib.ExtractResource(c.Node().Id())
	})
	wantCharts := []string{
		"TestingApp",
		"sdk.BlockExplorer",
		"sdk.HelmChart#otterscan",
		"sdk.Namespace",
		"sdk.HelmChart",
		"sdk.ClientSideApply",
	}
	is.Equal(wantCharts, gotCharts)
	is.Len(*app.Charts(), len(wantCharts))

	// Debug: Print all chart IDs and types
	for i, c := range *app.Charts() {
		id := crib.ExtractResource(c.Node().Id())
		chartType := "<unknown>"
		if dry.FromPtr(cdk8s.Chart_IsChart(c)) {
			chartType = "Chart"
		}
		t.Logf("Chart[%d]: ID=%s, Type=%s", i, id, chartType)
	}

	var otterscan, waitFor cdk8s.Chart
	for _, c := range *app.Charts() {
		if !dry.FromPtr(cdk8s.Chart_IsChart(c)) {
			continue
		}

		switch crib.ExtractResource(c.Node().Id()) {
		case "sdk.HelmChart#otterscan":
			otterscan = c
		case "sdk.ClientSideApply":
			waitFor = c
		}
	}

	// Assert that all expected charts are found
	is.NotNil(otterscan, "otterscan chart should not be nil")
	is.NotNil(waitFor, "waitFor chart should not be nil")

	t.Run("Otterscan", func(t *testing.T) {
		children := otterscan.Node().Children()
		t.Logf("otterscan chart children: %d", len(*children))
		child := otterscan.Node().DefaultChild()
		if child == nil {
			t.Log("otterscan chart's DefaultChild is nil; skipping ApiObject_Of check")
			return
		}
		var obj cdk8s.ApiObject
		is.NotPanics(func() {
			obj = cdk8s.ApiObject_Of(child)
		})
		is.NotNil(obj)

		is.Equal("helm.toolkit.fluxcd.io/v2beta1", *obj.ApiVersion())
		is.Equal("HelmRelease", *obj.Kind())
		is.Equal("test-blockexplorer", *obj.Metadata().Name())
		is.Equal("test-namespace", *obj.Metadata().Namespace())
	})

	t.Run("WaitFor", func(t *testing.T) {
		var obj cdk8s.ApiObject
		is.NotPanics(func() {
			obj = cdk8s.ApiObject_Of(waitFor.Node().DefaultChild())
		})
		is.NotNil(obj)

		is.Equal("crib.smartcontract.com/v1alpha1", *obj.ApiVersion())
		is.Equal("ClientSideApply", *obj.Kind())
		is.Equal("crib.smartcontract.com", *obj.ApiGroup())
		is.Equal("test-namespace", *obj.Metadata().Namespace())

		json := dry.As[map[string]any](obj.ToJson())
		is.NotNil(json)
		spec := dry.As[map[string]any](json["spec"])
		is.NotNil(spec)

		want := map[string]any{
			"onFailure": domain.FailureAbort,
			"action":    domain.ActionKubectl,
			"args": []any{
				"wait",
				"-n", "test-namespace",
				"--for=condition=ready",
				"pod",
				"-l=app.kubernetes.io/component=test-blockexplorer,app.kubernetes.io/name=devspace-app",
				"--timeout=600s",
			},
		}
		is.Equal(want, spec)
	})
}
