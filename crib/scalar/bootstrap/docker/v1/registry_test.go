package dockerv1

import (
	"context"
	"testing"

	"github.com/cdk8s-team/cdk8s-core-go/cdk8s/v2"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/crib-sdk/crib"
	"github.com/smartcontractkit/crib-sdk/internal"
	"github.com/smartcontractkit/crib-sdk/internal/core/common/dry"
)

func TestRegistryComponent(t *testing.T) {
	t.Parallel()
	internal.JSIIKernelMutex.Lock()
	t.Cleanup(internal.JSIIKernelMutex.Unlock)

	is := assert.New(t)

	app := internal.NewTestApp(t)
	ctx := internal.ContextWithConstruct(t.Context(), app.Chart)

	component, err := Component("test-registry", "5001")(ctx)
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

	obj := cdk8s.ApiObject_Of(apply.Node().DefaultChild())

	is.Equal("crib.smartcontract.com/v1alpha1", *obj.ApiVersion())
	is.Equal("ClientSideApply", *obj.Kind())
	is.Equal("crib.smartcontract.com", *obj.ApiGroup())
	is.Equal("default", *obj.Metadata().Namespace())

	json := dry.As[map[string]any](obj.ToJson())
	is.NotNil(json)
	spec := dry.As[map[string]any](json["spec"])
	is.NotNil(spec)

	want := map[string]any{
		"onFailure": "abort",
		"action":    "cmd",
		"args": []any{
			"REGISTRY_NAME=test-registry",
			"REGISTRY_PORT=5001",
			scriptPath,
		},
	}
	is.Equal(want, spec)
}

func TestProps_Validate(t *testing.T) {
	v, err := internal.NewValidator()
	require.NoError(t, err)
	ctx := internal.ContextWithValidator(context.Background(), v)

	tests := []struct {
		name    string
		props   Props
		wantErr bool
	}{
		{
			name: "valid props",
			props: Props{
				Name: "test-registry",
				Port: "5001",
			},
			wantErr: false,
		},
		{
			name: "empty name",
			props: Props{
				Name: "",
				Port: "5001",
			},
			wantErr: true,
		},
		{
			name: "empty port",
			props: Props{
				Name: "test-registry",
				Port: "",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.props.Validate(ctx)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
