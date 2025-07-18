package rolev1

import (
	"context"
	"testing"

	"github.com/cdk8s-team/cdk8s-plus-go/cdk8splus30/v2/k8s"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/crib-sdk/internal"
	"github.com/smartcontractkit/crib-sdk/internal/core/common/dry"
)

func TestNewRole_ValidProps(t *testing.T) {
	t.Parallel()
	internal.JSIIKernelMutex.Lock()
	defer internal.JSIIKernelMutex.Unlock()

	is := assert.New(t)
	must := require.New(t)

	props := &Props{
		Namespace:   "test-namespace",
		AppName:     "test-app",
		AppInstance: "test-instance",
		Name:        "test-role",
		Rules: []*k8s.PolicyRule{
			{
				ApiGroups: &[]*string{dry.ToPtr("")},
				Resources: &[]*string{dry.ToPtr("pods")},
				Verbs:     &[]*string{dry.ToPtr("get"), dry.ToPtr("list")},
			},
		},
	}

	app := internal.NewTestApp(t)
	ctx := internal.ContextWithConstruct(t.Context(), app.Chart)
	role, err := New(ctx, props)
	must.NoError(err)
	is.Equal("test-role", role.Name())
}

func TestProps_Validate(t *testing.T) {
	props := &Props{}
	ctx := context.Background()
	if err := props.Validate(ctx); err == nil {
		t.Error("expected validation error for missing required fields")
	}
}
