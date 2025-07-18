package domain

import (
	"fmt"
	"strings"

	"github.com/cdk8s-team/cdk8s-plus-go/cdk8splus30/v2/k8s"

	"github.com/smartcontractkit/crib-sdk/internal/core/common/dry"
)

// ContainerPort represents a port in a container.
type ContainerPort struct {
	Name          string
	Protocol      string
	ContainerPort int
}

// Container represents a container in a pod.
type Container struct {
	Name            string
	Image           string
	ImagePullPolicy string
	Ports           []ContainerPort
	Env             []EnvVar
	Resources       *Resources
	Command         []string
	Args            []string
	VolumeMounts    []VolumeMount
	UserID          int `validate:"required,min=1"`
	GroupID         int `validate:"required,min=1"`
}

// EnvVar represents an environment variable in a container.
type EnvVar struct {
	Name  string
	Value string
}

// Resources represents the resource requirements for a container.
type Resources struct {
	Limits   map[string]string
	Requests map[string]string
}

// VolumeMount represents a volume mount in a container.
type VolumeMount struct {
	Name      string
	MountPath string
	ReadOnly  bool
}

type ImageURI struct {
	Repository string
	Tag        string
	Digest     string
}

// ParseImageURI breaks down the Image URI string and returns an ImageURI struct.
func ParseImageURI(imageURI string) (ImageURI, error) {
	if imageURI == "" {
		return ImageURI{}, fmt.Errorf("image URI cannot be empty")
	}

	// Basic validation for obviously invalid URIs
	if strings.Contains(imageURI, "://") {
		return ImageURI{}, fmt.Errorf("invalid image URI format: %s", imageURI)
	}

	var result ImageURI

	// Handle digest format (repository@digest)
	if at := strings.LastIndex(imageURI, "@"); at != -1 {
		result.Repository = imageURI[:at]
		result.Digest = imageURI[at+1:]
		return result, nil
	}

	// Handle tag format (repository:tag)
	if colon := strings.LastIndex(imageURI, ":"); colon != -1 {
		beforeColon := imageURI[:colon]
		afterColon := imageURI[colon+1:]

		// Check if this is a port in a registry (contains slash after colon)
		// If there's a slash in the part after the colon, it's registry:port/image format
		if strings.Contains(afterColon, "/") {
			// This is registry:port/image format, no tag
			result.Repository = imageURI
			result.Tag = "latest"
		} else {
			// This is repository:tag format
			result.Repository = beforeColon
			result.Tag = afterColon
		}
	} else {
		// No tag specified, default to latest
		result.Repository = imageURI
		result.Tag = "latest"
	}

	return result, nil
}

// ConvertContainers converts the Container slice to k8s.Container slice.
func ConvertContainers(containers []*Container) *[]*k8s.Container {
	if len(containers) == 0 {
		return nil
	}

	k8sContainers := make([]*k8s.Container, len(containers))
	for i, c := range containers {
		container := &k8s.Container{
			Name:  dry.ToPtr(c.Name),
			Image: dry.ToPtr(c.Image),
		}

		if c.ImagePullPolicy != "" {
			container.ImagePullPolicy = dry.ToPtr(c.ImagePullPolicy)
		}
		if len(c.Command) > 0 {
			container.Command = dry.PtrSlice(c.Command)
		}
		if len(c.Args) > 0 {
			container.Args = dry.PtrSlice(c.Args)
		}

		if len(c.Ports) > 0 {
			containerPorts := make([]*k8s.ContainerPort, len(c.Ports))
			for j, p := range c.Ports {
				containerPorts[j] = &k8s.ContainerPort{
					Name:          dry.ToPtr(p.Name),
					ContainerPort: dry.ToPtr(float64(p.ContainerPort)),
					Protocol:      dry.ToPtr(p.Protocol),
				}
			}
			container.Ports = dry.ToPtr(containerPorts)
		}

		if len(c.Env) > 0 {
			envVars := make([]*k8s.EnvVar, len(c.Env))
			for j, e := range c.Env {
				envVars[j] = &k8s.EnvVar{
					Name:  dry.ToPtr(e.Name),
					Value: dry.ToPtr(e.Value),
				}
			}
			container.Env = dry.ToPtr(envVars)
		}

		if len(c.VolumeMounts) > 0 {
			volumeMounts := make([]*k8s.VolumeMount, len(c.VolumeMounts))
			for j, vm := range c.VolumeMounts {
				volumeMounts[j] = &k8s.VolumeMount{
					Name:      dry.ToPtr(vm.Name),
					MountPath: dry.ToPtr(vm.MountPath),
					ReadOnly:  dry.ToPtr(vm.ReadOnly),
				}
			}
			container.VolumeMounts = dry.ToPtr(volumeMounts)
		}

		if c.Resources != nil {
			container.Resources = &k8s.ResourceRequirements{
				Limits:   dry.ToPtr(ConvertResourceMap(c.Resources.Limits)),
				Requests: dry.ToPtr(ConvertResourceMap(c.Resources.Requests)),
			}
		}

		container.SecurityContext = &k8s.SecurityContext{
			RunAsUser:    dry.ToPtr(float64(c.UserID)),
			RunAsGroup:   dry.ToPtr(float64(c.GroupID)),
			RunAsNonRoot: dry.ToPtr(true),
		}

		k8sContainers[i] = container
	}

	return dry.ToPtr(k8sContainers)
}

// ConvertResourceMap converts a string map to a Quantity map.
func ConvertResourceMap(resourceMap map[string]string) map[string]k8s.Quantity {
	if len(resourceMap) == 0 {
		return nil
	}
	result := make(map[string]k8s.Quantity)
	for k, v := range resourceMap {
		result[k] = k8s.Quantity_FromString(&v)
	}
	return result
}
