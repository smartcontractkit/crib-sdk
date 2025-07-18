package fakev1

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/crib-sdk/internal"
)

func TestFakeComponent_Validate_RequiredFields(t *testing.T) {
	v, err := internal.NewValidator()
	require.NoError(t, err)
	ctx := internal.ContextWithValidator(context.Background(), v)

	tests := []struct {
		name    string
		props   *Props
		wantErr bool
	}{
		{
			name: "valid props",
			props: &Props{
				Namespace:       "test-namespace",
				AppInstanceName: "test-instance",
				Image:           "test-image:latest",
				EnvVars: map[string]string{
					"TEST_VAR": "test-value",
				},
				Ports: []ContainerPort{
					{
						Name:          "http",
						ContainerPort: 8080,
						Protocol:      "TCP",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "missing namespace",
			props: &Props{
				AppInstanceName: "test-instance",
				Image:           "test-image:latest",
			},
			wantErr: true,
		},
		{
			name: "missing app instance name",
			props: &Props{
				Namespace: "test-namespace",
				Image:     "test-image:latest",
			},
			wantErr: true,
		},
		{
			name: "missing image",
			props: &Props{
				Namespace:       "test-namespace",
				AppInstanceName: "test-instance",
			},
			wantErr: true,
		},
		{
			name: "invalid port",
			props: &Props{
				Namespace:       "test-namespace",
				AppInstanceName: "test-instance",
				Image:           "test-image:latest",
				Ports: []ContainerPort{
					{
						Name:          "invalid-port",
						ContainerPort: 0, // Invalid port
						Protocol:      "TCP",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid protocol",
			props: &Props{
				Namespace:       "test-namespace",
				AppInstanceName: "test-instance",
				Image:           "test-image:latest",
				Ports: []ContainerPort{
					{
						Name:          "http",
						ContainerPort: 8080,
						Protocol:      "INVALID", // Invalid protocol
					},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid image pull policy",
			props: &Props{
				Namespace:       "test-namespace",
				AppInstanceName: "test-instance",
				Image:           "test-image:latest",
				ImagePullPolicy: "Invalid", // Invalid pull policy
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

func TestFakeComponent_Creation(t *testing.T) {
	internal.JSIIKernelMutex.Lock()
	t.Cleanup(internal.JSIIKernelMutex.Unlock)

	app := internal.NewTestApp(t)
	ctx := internal.ContextWithConstruct(t.Context(), app.Chart)

	props := &Props{
		Namespace:       "test-namespace",
		AppInstanceName: "test-fake",
		Image:           "fake:latest",
		Command:         []string{"/bin/sh"},
		Args:            []string{"-c", "echo 'Hello World'"},
		EnvVars: map[string]string{
			"ENV_VAR1": "value1",
			"ENV_VAR2": "value2",
		},
		Ports: []ContainerPort{
			{
				Name:          "http",
				ContainerPort: 8080,
				Protocol:      "TCP",
			},
			{
				Name:          "metrics",
				ContainerPort: 9090,
				Protocol:      "TCP",
			},
		},
		Replicas: 2,
	}

	component, err := Component(props)(ctx)
	require.NoError(t, err)
	require.NotNil(t, component)

	result, ok := component.(Result)
	require.True(t, ok)
	assert.Equal(t, "test-fake", result.nodeName)
	assert.Equal(t, "test-namespace", result.namespace)
}

func TestConvertToK8sPorts(t *testing.T) {
	ports := []ContainerPort{
		{
			Name:          "http",
			ContainerPort: 8080,
			Protocol:      "TCP",
		},
		{
			Name:          "grpc",
			ContainerPort: 9090,
			Protocol:      "UDP",
		},
	}

	k8sPorts := convertToK8sPorts(ports)
	require.NotNil(t, k8sPorts)
	require.Len(t, *k8sPorts, 2)

	httpPort := (*k8sPorts)[0]
	assert.Equal(t, "http", *httpPort.Name)
	assert.Equal(t, float64(8080), *httpPort.ContainerPort)
	assert.Equal(t, "TCP", *httpPort.Protocol)

	grpcPort := (*k8sPorts)[1]
	assert.Equal(t, "grpc", *grpcPort.Name)
	assert.Equal(t, float64(9090), *grpcPort.ContainerPort)
	assert.Equal(t, "UDP", *grpcPort.Protocol)
}

func TestFakeComponent_Defaults(t *testing.T) {
	v, err := internal.NewValidator()
	require.NoError(t, err)
	ctx := internal.ContextWithValidator(context.Background(), v)

	props := &Props{
		Namespace:       "test-namespace",
		AppInstanceName: "test-instance",
		Image:           "test-image:latest",
	}

	// Test that validation applies defaults
	err = props.Validate(ctx)
	require.NoError(t, err)

	// ImagePullPolicy should have default value
	assert.Equal(t, "IfNotPresent", props.ImagePullPolicy)
	// Replicas should have default value
	assert.Equal(t, int32(1), props.Replicas)
}
