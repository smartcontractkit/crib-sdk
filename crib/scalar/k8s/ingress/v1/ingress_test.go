package ingressv1

import (
	"testing"

	"github.com/cdk8s-team/cdk8s-plus-go/cdk8splus30/v2/k8s"
	"github.com/stretchr/testify/assert"

	"github.com/smartcontractkit/crib-sdk/crib"
	"github.com/smartcontractkit/crib-sdk/internal"
	"github.com/smartcontractkit/crib-sdk/internal/core/common/dry"
)

func TestNewIngress(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		props     *Props
		expectErr assert.ErrorAssertionFunc
	}{
		{
			name:      "Valid Props with Rules",
			props:     validProps(),
			expectErr: assert.NoError,
		},
		{
			name: "Invalid Props with missing Name",
			props: &Props{
				Namespace: "production",
				Rules: []IngressRule{
					{
						Host:     "example.com",
						PathType: "Prefix",
						Path:     "/api",
						Service: IngressBackendService{
							Name: "api-service",
							Port: 80,
						},
					},
				},
			},
			expectErr: assert.Error,
		},
		{
			name: "Invalid Props with missing Rules",
			props: &Props{
				Name:      "api-ingress",
				Namespace: "production",
				Rules:     []IngressRule{},
			},
			expectErr: assert.Error,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			internal.JSIIKernelMutex.Lock()
			defer internal.JSIIKernelMutex.Unlock()

			is := assert.New(t)

			app := internal.NewTestApp(t)
			ctx := internal.ContextWithConstruct(t.Context(), app.Chart)

			component, err := New(ctx, tt.props)

			if tt.expectErr(t, err) && err != nil {
				is.Nil(component)
			} else {
				is.NoError(err)
				is.NotNil(component)
				is.Implements((*crib.Component)(nil), component)
				internal.SynthAndSnapYamls(t, app)
			}
		})
	}
}

func validProps() *Props {
	return &Props{
		Namespace:   "test-namespace",
		AppName:     "test-app",
		AppInstance: "test-app-123",
		Name:        "api-ingress",
		Annotations: map[string]*string{
			"nginx.ingress.kubernetes.io/rewrite-target": dry.ToPtr("/"),
		},
		IngressClassName: "nginx",
		Rules: []IngressRule{
			{
				Host:     "example.com",
				PathType: "Prefix",
				Path:     "/api",
				Service: IngressBackendService{
					Name: "api-service",
					Port: 80,
				},
			},
		},
	}
}

func TestNewIngress_CustomizeCreatedScalar(t *testing.T) {
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

	ingress := dry.MustAs[k8s.KubeIngress](component.Component)

	// Customize properties of resulting scalar using cdk8plus API
	ingress.Metadata().AddLabel(dry.ToPtr("app"), dry.ToPtr("api"))

	internal.SynthAndSnapYamls(t, app)
}
