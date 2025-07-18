package domain

import (
	"github.com/cdk8s-team/cdk8s-plus-go/cdk8splus30/v2/k8s"

	"github.com/smartcontractkit/crib-sdk/internal/core/common/dry"
)

// Volume represents a volume in a pod.
type Volume struct {
	Name string
	// Optional: ConfigMapRef is the name of the configmap to mount.
	ConfigMapRef string
}

// PersistentVolumeClaim represents a persistent volume claim template.
type PersistentVolumeClaim struct {
	Capacity   string `validate:"required"`
	NameSuffix string `validate:"required"`
}

// ConvertVolumes converts the Volume slice to k8s.Volume slice.
func ConvertVolumes(volumes []*Volume) *[]*k8s.Volume {
	if len(volumes) == 0 {
		return nil
	}

	k8sVolumes := make([]*k8s.Volume, len(volumes))
	for i, v := range volumes {
		if v.ConfigMapRef != "" {
			k8sVolumes[i] = &k8s.Volume{
				Name: dry.ToPtr(v.Name),
				ConfigMap: &k8s.ConfigMapVolumeSource{
					Name: dry.ToPtr(v.ConfigMapRef),
				},
			}
		} else {
			k8sVolumes[i] = &k8s.Volume{
				Name: dry.ToPtr(v.Name),
			}
		}
		// Note: This is simplified and would need expansion based on actual volume source types
	}

	return dry.ToPtr(k8sVolumes)
}
