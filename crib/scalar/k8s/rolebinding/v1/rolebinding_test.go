package v1

import (
	"context"
	"testing"

	"github.com/cdk8s-team/cdk8s-plus-go/cdk8splus30/v2/k8s"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/crib-sdk/crib"
	"github.com/smartcontractkit/crib-sdk/internal"
	"github.com/smartcontractkit/crib-sdk/internal/core/common/dry"
)

func TestRoleBinding_Validate(t *testing.T) {
	t.Parallel()
	internal.JSIIKernelMutex.Lock()
	defer internal.JSIIKernelMutex.Unlock()

	props := &Props{
		Namespace:   "default",
		AppName:     "test-app",
		AppInstance: "test-instance",
		Name:        "test-rolebinding",
		RoleRef:     &k8s.RoleRef{Kind: dry.ToPtr("Role"), Name: dry.ToPtr("test-role")},
		Subjects:    []*k8s.Subject{{Kind: dry.ToPtr("User"), Name: dry.ToPtr("test-user")}},
	}
	ctx := context.Background()
	if err := props.Validate(ctx); err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestNewRoleBinding(t *testing.T) {
	t.Parallel()
	internal.JSIIKernelMutex.Lock()
	t.Cleanup(internal.JSIIKernelMutex.Unlock)

	props := &Props{
		Namespace:   "default",
		AppName:     "test-app",
		AppInstance: "test-instance",
		Name:        "test-rolebinding",
		RoleRef: &k8s.RoleRef{
			ApiGroup: dry.ToPtr("rbac.authorization.k8s.io"),
			Kind:     dry.ToPtr("Role"),
			Name:     dry.ToPtr("test-role"),
		},
		Subjects: []*k8s.Subject{{
			Kind:     dry.ToPtr("User"),
			Name:     dry.ToPtr("test-user"),
			ApiGroup: dry.ToPtr("rbac.authorization.k8s.io"),
		}},
	}
	app := internal.NewTestApp(t)
	ctx := internal.ContextWithConstruct(t.Context(), app.Chart)
	roleBinding, err := New(ctx, crib.Props(props))

	must := require.New(t)

	must.NoError(err)
	must.Equal("test-rolebinding", roleBinding.Name())
}
