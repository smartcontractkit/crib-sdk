package servicev1

import (
	"testing"

	"github.com/aws/jsii-runtime-go"
	"github.com/cdk8s-team/cdk8s-plus-go/cdk8splus30/v2/k8s"
	"github.com/stretchr/testify/assert"

	"github.com/smartcontractkit/crib-sdk/crib"
	"github.com/smartcontractkit/crib-sdk/internal"
	"github.com/smartcontractkit/crib-sdk/internal/core/common/dry"

	syncCDK8s "github.com/smartcontractkit/crib-sdk/internal/cdk8s"
)

var port8080 = syncCDK8s.IntOrStringFromNumber(dry.ToPtr[float64](8080))

func TestNewService(t *testing.T) {
	tests := []struct {
		name      string
		props     *Props
		expectErr assert.ErrorAssertionFunc
	}{
		{
			name:      "Valid Props with Ports and Selector",
			props:     validProps(),
			expectErr: assert.NoError,
		},
		{
			name: "Invalid Props with missing Name",
			props: &Props{
				Namespace: "production",
				Ports: []*k8s.ServicePort{
					{
						Port:       jsii.Number(80),
						TargetPort: port8080,
					},
				},
				Selector: map[string]*string{
					"app": dry.ToPtr("web"),
				},
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
				return
			}
			is.NoError(err)
			is.NotNil(component)
			is.Implements((*crib.Component)(nil), component)
			internal.SynthAndSnapYamls(t, app)
		})
	}
}

func validProps() *Props {
	return &Props{
		Namespace:   "production",
		AppName:     "test-app",
		AppInstance: "test-app-123",
		Name:        "web-service",
		Selector: map[string]*string{
			"app": dry.ToPtr("web"),
		},
		ServiceType: "ClusterIP",
		Ports: []*k8s.ServicePort{
			{
				Port:       jsii.Number(80),
				TargetPort: port8080,
			},
		},
	}
}

func TestNewService_CustomizeCreatedScalar(t *testing.T) {
	t.Parallel()
	internal.JSIIKernelMutex.Lock()
	defer internal.JSIIKernelMutex.Unlock()

	is := assert.New(t)

	app := internal.NewTestApp(t)
	ctx := internal.ContextWithConstruct(t.Context(), app.Chart)

	// Example 2: Props with Ports and Selector
	testProps := validProps()
	component, err := New(ctx, testProps)
	is.NoError(err)
	is.NotNil(component)
	is.Implements((*crib.Component)(nil), component)

	serviceScalar := dry.MustAs[*Service](component)

	// I can customize properties of resulting serviceScalar using cdk8plus API
	kubeService := dry.MustAs[k8s.KubeService](serviceScalar.Component)
	kubeService.Metadata().AddAnnotation(dry.ToPtr("foo"), dry.ToPtr("bar"))

	internal.SynthAndSnapYamls(t, app)
}
