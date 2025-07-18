package secretv1

import (
	"testing"

	"github.com/cdk8s-team/cdk8s-plus-go/cdk8splus30/v2/k8s"
	"github.com/stretchr/testify/assert"

	"github.com/smartcontractkit/crib-sdk/crib"
	"github.com/smartcontractkit/crib-sdk/internal"
	"github.com/smartcontractkit/crib-sdk/internal/core/common/dry"
)

func TestNewSecret(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		props     *Props
		expectErr assert.ErrorAssertionFunc
	}{
		{
			name:      "Valid Props with Data",
			props:     validProps(),
			expectErr: assert.NoError,
		},
		{
			name: "Invalid Props with missing Name",
			props: &Props{
				Namespace: "production",
				Type:      "Opaque",
				StringData: map[string]*string{
					"username": dry.ToPtr("admin"),
				},
			},
			expectErr: assert.Error,
		},
		{
			name: "Invalid Props with missing Data",
			props: &Props{
				Name:       "test-secret",
				Namespace:  "production",
				Type:       "Opaque",
				StringData: map[string]*string{},
			},
			expectErr: assert.NoError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			internal.JSIIKernelMutex.Lock()
			t.Cleanup(internal.JSIIKernelMutex.Unlock)

			is := assert.New(t)

			app := internal.NewTestApp(t)
			ctx := internal.ContextWithConstruct(t.Context(), app.Chart)

			component, err := New(ctx, tt.props)

			if tt.expectErr(t, err) && err != nil {
				is.Nil(component)
				return
			}

			is.NoError(err)
			is.NotNil(component)
			is.Implements((*crib.Component)(nil), component)
			internal.SynthAndSnapYamls(t, app)
		})
	}
}

func TestNewSecret_CustomizeCreatedScalar(t *testing.T) {
	t.Parallel()
	internal.JSIIKernelMutex.Lock()
	defer internal.JSIIKernelMutex.Unlock()

	is := assert.New(t)

	app := internal.NewTestApp(t)
	ctx := internal.ContextWithConstruct(t.Context(), app.Chart)

	// Example with additional customization
	testProps := validProps()
	component, err := New(ctx, testProps)
	is.NoError(err)
	is.NotNil(component)
	is.Implements((*crib.Component)(nil), component)

	res := dry.MustAs[*Secret](component)
	secret := dry.MustAs[k8s.KubeSecret](res.Component)

	// Customize properties of resulting scalar using cdk8plus API
	secret.Metadata().AddLabel(dry.ToPtr("app"), dry.ToPtr("api"))

	internal.SynthAndSnapYamls(t, app)
}

func validProps() *Props {
	return &Props{
		Name:      "test-secret",
		Namespace: "production",
		Type:      "Opaque",
		StringData: map[string]*string{
			"username": dry.ToPtr("admin"),
			"password": dry.ToPtr("secret"),
			"config":   dry.ToPtr("some-config-value"),
		},
		Immutable: dry.ToPtr(true),
	}
}
