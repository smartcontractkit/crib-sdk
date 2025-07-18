package domain

import (
	"testing"

	"github.com/cdk8s-team/cdk8s-plus-go/cdk8splus30/v2/k8s"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvertVolumes(t *testing.T) {
	tests := []struct {
		name     string
		volumes  []*Volume
		expected int // expected number of volumes
		validate func(t *testing.T, result *[]*k8s.Volume)
	}{
		{
			name:     "nil volumes slice",
			volumes:  nil,
			expected: 0,
			validate: func(t *testing.T, result *[]*k8s.Volume) {
				assert.Nil(t, result, "result should be nil for nil input")
			},
		},
		{
			name:     "empty volumes slice",
			volumes:  []*Volume{},
			expected: 0,
			validate: func(t *testing.T, result *[]*k8s.Volume) {
				assert.Nil(t, result, "result should be nil for empty slice")
			},
		},
		{
			name: "single volume with ConfigMapRef",
			volumes: []*Volume{
				{
					Name:         "config-volume",
					ConfigMapRef: "my-configmap",
				},
			},
			expected: 1,
			validate: func(t *testing.T, result *[]*k8s.Volume) {
				require.NotNil(t, result, "result should not be nil")
				require.Len(t, *result, 1, "should have exactly one volume")

				volume := (*result)[0]
				require.NotNil(t, volume, "volume should not be nil")
				require.NotNil(t, volume.Name, "volume name should not be nil")
				assert.Equal(t, "config-volume", *volume.Name, "volume name should match")

				require.NotNil(t, volume.ConfigMap, "ConfigMap should not be nil")
				require.NotNil(t, volume.ConfigMap.Name, "ConfigMap name should not be nil")
				assert.Equal(t, "my-configmap", *volume.ConfigMap.Name, "ConfigMap name should match")
			},
		},
		{
			name: "single volume without ConfigMapRef",
			volumes: []*Volume{
				{
					Name: "empty-volume",
				},
			},
			expected: 1,
			validate: func(t *testing.T, result *[]*k8s.Volume) {
				require.NotNil(t, result, "result should not be nil")
				require.Len(t, *result, 1, "should have exactly one volume")

				volume := (*result)[0]
				require.NotNil(t, volume, "volume should not be nil")
				require.NotNil(t, volume.Name, "volume name should not be nil")
				assert.Equal(t, "empty-volume", *volume.Name, "volume name should match")
				assert.Nil(t, volume.ConfigMap, "ConfigMap should be nil when ConfigMapRef is empty")
			},
		},
		{
			name: "multiple volumes with mixed ConfigMapRef",
			volumes: []*Volume{
				{
					Name:         "config-volume-1",
					ConfigMapRef: "configmap-1",
				},
				{
					Name: "empty-volume",
				},
				{
					Name:         "config-volume-2",
					ConfigMapRef: "configmap-2",
				},
			},
			expected: 3,
			validate: func(t *testing.T, result *[]*k8s.Volume) {
				require.NotNil(t, result, "result should not be nil")
				require.Len(t, *result, 3, "should have exactly three volumes")

				volumes := *result

				// First volume with ConfigMap
				volume1 := volumes[0]
				require.NotNil(t, volume1, "first volume should not be nil")
				require.NotNil(t, volume1.Name, "first volume name should not be nil")
				assert.Equal(t, "config-volume-1", *volume1.Name, "first volume name should match")
				require.NotNil(t, volume1.ConfigMap, "first volume ConfigMap should not be nil")
				require.NotNil(t, volume1.ConfigMap.Name, "first volume ConfigMap name should not be nil")
				assert.Equal(t, "configmap-1", *volume1.ConfigMap.Name, "first volume ConfigMap name should match")

				// Second volume without ConfigMap
				volume2 := volumes[1]
				require.NotNil(t, volume2, "second volume should not be nil")
				require.NotNil(t, volume2.Name, "second volume name should not be nil")
				assert.Equal(t, "empty-volume", *volume2.Name, "second volume name should match")
				assert.Nil(t, volume2.ConfigMap, "second volume ConfigMap should be nil")

				// Third volume with ConfigMap
				volume3 := volumes[2]
				require.NotNil(t, volume3, "third volume should not be nil")
				require.NotNil(t, volume3.Name, "third volume name should not be nil")
				assert.Equal(t, "config-volume-2", *volume3.Name, "third volume name should match")
				require.NotNil(t, volume3.ConfigMap, "third volume ConfigMap should not be nil")
				require.NotNil(t, volume3.ConfigMap.Name, "third volume ConfigMap name should not be nil")
				assert.Equal(t, "configmap-2", *volume3.ConfigMap.Name, "third volume ConfigMap name should match")
			},
		},
		{
			name: "volume with empty string ConfigMapRef",
			volumes: []*Volume{
				{
					Name:         "test-volume",
					ConfigMapRef: "",
				},
			},
			expected: 1,
			validate: func(t *testing.T, result *[]*k8s.Volume) {
				require.NotNil(t, result, "result should not be nil")
				require.Len(t, *result, 1, "should have exactly one volume")

				volume := (*result)[0]
				require.NotNil(t, volume, "volume should not be nil")
				require.NotNil(t, volume.Name, "volume name should not be nil")
				assert.Equal(t, "test-volume", *volume.Name, "volume name should match")
				assert.Nil(t, volume.ConfigMap, "ConfigMap should be nil for empty string ConfigMapRef")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ConvertVolumes(tt.volumes)
			tt.validate(t, result)
		})
	}
}

func TestVolume_Struct(t *testing.T) {
	t.Run("volume creation", func(t *testing.T) {
		volume := &Volume{
			Name:         "test-volume",
			ConfigMapRef: "test-configmap",
		}

		assert.Equal(t, "test-volume", volume.Name, "volume name should be set correctly")
		assert.Equal(t, "test-configmap", volume.ConfigMapRef, "ConfigMapRef should be set correctly")
	})

	t.Run("volume with empty ConfigMapRef", func(t *testing.T) {
		volume := &Volume{
			Name: "test-volume",
		}

		assert.Equal(t, "test-volume", volume.Name, "volume name should be set correctly")
		assert.Empty(t, volume.ConfigMapRef, "ConfigMapRef should be empty by default")
	})
}

func TestPersistentVolumeClaim_Struct(t *testing.T) {
	t.Run("PVC creation", func(t *testing.T) {
		pvc := &PersistentVolumeClaim{
			Capacity:   "10Gi",
			NameSuffix: "data",
		}

		assert.Equal(t, "10Gi", pvc.Capacity, "capacity should be set correctly")
		assert.Equal(t, "data", pvc.NameSuffix, "name suffix should be set correctly")
	})
}
