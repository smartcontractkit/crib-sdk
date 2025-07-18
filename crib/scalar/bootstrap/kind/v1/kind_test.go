package kindv1

import (
	"context"
	"fmt"
	"testing"

	"github.com/cdk8s-team/cdk8s-core-go/cdk8s/v2"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/crib-sdk/crib"
	"github.com/smartcontractkit/crib-sdk/internal"
	"github.com/smartcontractkit/crib-sdk/internal/core/common/dry"
)

type invalidProps struct {
	Invalid string
}

func (p *invalidProps) Validate(ctx context.Context) error {
	return nil
}

func TestNew(t *testing.T) {
	t.Parallel()
	internal.JSIIKernelMutex.Lock()
	t.Cleanup(internal.JSIIKernelMutex.Unlock)

	is := assert.New(t)

	app := internal.NewTestApp(t)
	ctx := internal.ContextWithConstruct(t.Context(), app.Chart)

	component, err := Component("test-cluster")(ctx)
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
			fmt.Sprintf("KIND_CONFIG_FILE=%s", defaultsPath),
			"KIND_CLUSTER_NAME=test-cluster",
			scriptPath,
		},
	}
	is.Equal(want, spec)
}

func TestProps_Validate(t *testing.T) {
	t.Parallel()

	v, err := internal.NewValidator()
	require.NoError(t, err)
	ctx := internal.ContextWithValidator(t.Context(), v)

	tests := []struct {
		name  string
		props *Props
		want  assert.ErrorAssertionFunc
	}{
		{
			name: "valid props",
			props: &Props{
				Name: "test-cluster",
			},
			want: assert.NoError,
		},
		{
			name: "empty name",
			props: &Props{
				Name: "",
			},
			want: assert.Error,
		},
		{
			name:  "nil props",
			props: nil,
			want:  assert.Error,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tt.want(t, tt.props.Validate(ctx))
		})
	}
}
