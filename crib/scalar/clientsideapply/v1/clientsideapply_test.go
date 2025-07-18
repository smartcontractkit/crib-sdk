package clientsideapplyv1

import (
	"testing"

	"github.com/cdk8s-team/cdk8s-core-go/cdk8s/v2"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"

	"github.com/smartcontractkit/crib-sdk/crib"
	"github.com/smartcontractkit/crib-sdk/internal"
	"github.com/smartcontractkit/crib-sdk/internal/core/common/dry"
)

func TestNewClientSideApply(t *testing.T) {
	t.Parallel()
	internal.JSIIKernelMutex.Lock()
	defer internal.JSIIKernelMutex.Unlock()
	is := assert.New(t)

	app := internal.NewTestApp(t)
	ctx := internal.ContextWithConstruct(t.Context(), app.Chart)

	testProps := &Props{
		Namespace: "test-namespace",
		OnFailure: "continue",
		Action:    "task",
		Args: []string{
			"go:lint",
		},
	}

	component, err := New(ctx, testProps)
	is.NoError(err)
	is.NotNil(component)

	gotCharts := lo.Map(*app.Charts(), func(c cdk8s.Chart, _ int) string {
		return crib.ExtractResource(c.Node().Id())
	})
	wantCharts := []string{
		"TestingApp",
		"sdk.ClientSideApply",
	}
	is.Equal(wantCharts, gotCharts)
	apply := (*app.Charts())[1]
	obj := cdk8s.ApiObject_Of(apply)

	is.Equal("crib.smartcontract.com/v1alpha1", *obj.ApiVersion())
	is.Equal("ClientSideApply", *obj.Kind())
	is.Equal("crib.smartcontract.com", *obj.ApiGroup())
	is.Equal("test-namespace", *obj.Metadata().Namespace())
	want := map[string]any{
		"onFailure": "continue",
		"action":    "task",
		"args": []any{
			"go:lint",
		},
	}
	is.Equal(want, dry.As[map[string]any](obj.ToJson())["spec"])

	internal.SynthAndSnapYamls(t, app)
}
