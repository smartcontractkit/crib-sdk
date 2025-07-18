package nginxcontrollerv1

import (
	"testing"

	"github.com/cdk8s-team/cdk8s-core-go/cdk8s/v2"
	"github.com/gkampitakis/go-snaps/snaps"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"

	"github.com/smartcontractkit/crib-sdk/crib"
	"github.com/smartcontractkit/crib-sdk/internal"
	"github.com/smartcontractkit/crib-sdk/internal/core/common/dry"
)

func TestComponent(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test in short mode.")
	}
	t.Parallel()
	internal.JSIIKernelMutex.Lock()
	defer internal.JSIIKernelMutex.Unlock()
	is := assert.New(t)

	app := internal.NewTestApp(t)
	ctx := internal.ContextWithConstruct(t.Context(), app.Chart)

	c := Component()
	component, err := c(ctx)
	is.NoError(err)
	is.NotNil(component)

	// Normalize the chart names into a more readable format.
	gotCharts := lo.Map(*app.Charts(), func(c cdk8s.Chart, _ int) string {
		return crib.ExtractResource(c.Node().Id())
	})
	wantCharts := []string{
		"TestingApp",
		"sdk.NginxController",
		"sdk.Namespace",
		"sdk.RemoteApply",
		"sdk.ClientSideApply",
	}
	is.Equal(wantCharts, gotCharts)
	is.Len(*app.Charts(), len(wantCharts))

	var ns, apply cdk8s.Chart
	for _, c := range *app.Charts() {
		if !dry.FromPtr(cdk8s.Chart_IsChart(c)) {
			continue
		}

		switch crib.ExtractResource(c.Node().Id()) {
		case "sdk.Namespace":
			ns = c
		case "sdk.ClientSideApply":
			apply = c
		}
	}

	t.Run("Namespace", func(t *testing.T) {
		var obj cdk8s.ApiObject
		is.NotPanics(func() {
			obj = cdk8s.ApiObject_Of(ns.Node().DefaultChild())
		})
		is.NotNil(obj)

		is.Equal("v1", *obj.ApiVersion())
		is.Equal("Namespace", *obj.Kind())
		is.Equal("ingress-nginx", *obj.Metadata().Name())
	})

	t.Run("ClientSideApply", func(t *testing.T) {
		var obj cdk8s.ApiObject
		is.NotPanics(func() {
			obj = cdk8s.ApiObject_Of(apply.Node().DefaultChild())
		})
		is.NotNil(obj)

		is.Equal("crib.smartcontract.com/v1alpha1", *obj.ApiVersion())
		is.Equal("ClientSideApply", *obj.Kind())
		is.Equal("crib.smartcontract.com", *obj.ApiGroup())
	})

	snaps.MatchSnapshot(t, *app.SynthYaml())
}
