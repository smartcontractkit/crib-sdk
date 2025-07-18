package statefulsetv1

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/smartcontractkit/crib-sdk/internal"
	"github.com/smartcontractkit/crib-sdk/internal/core/domain"
)

func TestComponent(t *testing.T) {
	t.Parallel()
	internal.JSIIKernelMutex.Lock()
	defer internal.JSIIKernelMutex.Unlock()

	is := assert.New(t)

	app := internal.NewTestApp(t)
	ctx := internal.ContextWithConstruct(t.Context(), app.Chart)

	// Valid props for testing
	validProps := validProps()

	t.Run("passes validation", func(t *testing.T) {
		component := Component(validProps)

		result, err := component(ctx)

		is.NoError(err)
		is.NotNil(result)
		is.IsType(&StatefulSetComposite{}, result)
	})

	t.Run("validation fails", func(t *testing.T) {
		// Invalid props - missing required fields
		invalidProps := &Props{}

		ctx := internal.ContextWithConstruct(t.Context(), app.Chart)
		component := Component(invalidProps)

		result, err := component(ctx)

		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

func validProps() *Props {
	validProps := &Props{
		Name:                "test-statefulset",
		Namespace:           "test-namespace",
		AppInstance:         "test-instance",
		AppName:             "test-app",
		Labels:              map[string]string{"app": "test-statefulset", "component": "database"},
		Annotations:         map[string]string{"description": "Test StatefulSet for unit tests"},
		PodManagementPolicy: "OrderedReady",
		Containers: []*domain.Container{
			{
				Name:            "main",
				Image:           "nginx:latest",
				ImagePullPolicy: "IfNotPresent",
				Ports: []domain.ContainerPort{
					{
						Name:          "http",
						ContainerPort: 80,
						Protocol:      "TCP",
					},
				},
				Env: []domain.EnvVar{
					{
						Name:  "POD_NAME",
						Value: "$(POD_NAME)",
					},
				},
				Resources: &domain.Resources{
					Limits:   map[string]string{"cpu": "200m", "memory": "256Mi"},
					Requests: map[string]string{"cpu": "100m", "memory": "128Mi"},
				},
				UserID:  1000,
				GroupID: 1000,
			},
		},
		Volumes: []*domain.Volume{
			{
				Name: "data-volume",
			},
		},
		VolumeClaimTemplates: []*domain.PersistentVolumeClaim{
			{
				NameSuffix: "data",
				Capacity:   "200m",
			},
		},
	}
	return validProps
}

func TestStatefulsetSynth(t *testing.T) {
	t.Parallel()
	internal.JSIIKernelMutex.Lock()
	defer internal.JSIIKernelMutex.Unlock()

	t.Run("creates statefulset with correct properties", func(t *testing.T) {
		// Create app & chart
		app := internal.NewTestApp(t)
		ctx := internal.ContextWithConstruct(t.Context(), app.Chart)

		// Create props with all fields populated
		props := validProps()
		// Create statefulset
		statefulset, err := statefulset(ctx, props)

		// Assert
		assert.NoError(t, err)
		assert.NotNil(t, statefulset)
		assert.Equal(t, props.Name, statefulset.GetName())
		assert.Equal(t, props.Namespace, statefulset.GetNamespace())
		assert.Equal(t, props.AppInstance, statefulset.GetAppInstance())

		internal.SynthAndSnapYamls(t, app)
	})
}
