package fakev1

import (
	"context"
	"fmt"

	"github.com/cdk8s-team/cdk8s-core-go/cdk8s/v2"
	"github.com/cdk8s-team/cdk8s-plus-go/cdk8splus30/v2/k8s"

	"github.com/smartcontractkit/crib-sdk/crib"
	"github.com/smartcontractkit/crib-sdk/internal"
	"github.com/smartcontractkit/crib-sdk/internal/core/common/dry"

	servicev1 "github.com/smartcontractkit/crib-sdk/crib/scalar/k8s/service/v1"
)

const ComponentName = "sdk.composite.fake.v1"

// ContainerPort represents a port configuration for the container.
type ContainerPort struct {
	Name          string `validate:"required,lte=63,dns_rfc1035_label"`
	Protocol      string `default:"TCP"                                validate:"oneof=TCP UDP"`
	ContainerPort int    `validate:"required,min=1,max=65535"`
}

// Props contains properties for the fake-ea component.
type Props struct {
	Namespace       string `validate:"required"`
	AppInstanceName string `validate:"required"`
	Image           string `validate:"required"`
	ImagePullPolicy string `default:"IfNotPresent" validate:"oneof=Always IfNotPresent Never"`
	Command         []string
	Args            []string
	EnvVars         map[string]string
	Ports           []ContainerPort `validate:"dive"`
	Replicas        int32           `default:"1"`
}

// Result represents the result of creating a fake-ea component.
type Result struct {
	crib.Component
	nodeName  string
	namespace string
}

// Validate validates the props.
func (p *Props) Validate(ctx context.Context) error {
	v := internal.ValidatorFromContext(ctx)
	return v.Struct(p)
}

// convertToK8sPorts converts our ContainerPort slice to k8s ContainerPort slice.
func convertToK8sPorts(ports []ContainerPort) *[]*k8s.ContainerPort {
	k8sPorts := make([]*k8s.ContainerPort, len(ports))
	for i, port := range ports {
		k8sPorts[i] = &k8s.ContainerPort{
			Name:          dry.ToPtr(port.Name),
			ContainerPort: dry.ToPtr(float64(port.ContainerPort)),
			Protocol:      dry.ToPtr(port.Protocol),
		}
	}
	return &k8sPorts
}

// Component returns a new fake composite component.
func Component(props *Props) crib.ComponentFunc {
	return func(ctx context.Context) (crib.Component, error) {
		if err := props.Validate(ctx); err != nil {
			return nil, err
		}
		return fake(ctx, props)
	}
}

// fake creates and returns a new fake composite component.
func fake(ctx context.Context, props crib.Props) (crib.Component, error) {
	parent := internal.ConstructFromContext(ctx)
	chart := cdk8s.NewChart(parent, crib.ResourceID(ComponentName, props), nil)
	ctx = internal.ContextWithConstruct(ctx, chart)

	fakeProps := dry.MustAs[*Props](props)

	// Create deployment labels
	labels := map[string]*string{
		"app.kubernetes.io/name":     dry.ToPtr("fake"),
		"app.kubernetes.io/instance": dry.ToPtr(fakeProps.AppInstanceName),
	}

	// Convert environment variables
	envVars := make([]*k8s.EnvVar, 0, len(fakeProps.EnvVars))
	for key, value := range fakeProps.EnvVars {
		envVars = append(envVars, &k8s.EnvVar{
			Name:  dry.ToPtr(key),
			Value: dry.ToPtr(value),
		})
	}

	// Create the deployment with pod security policy
	_ = k8s.NewKubeDeployment(chart, dry.ToPtr("fake-deployment"), &k8s.KubeDeploymentProps{
		Metadata: &k8s.ObjectMeta{
			Name:      dry.ToPtr(fakeProps.AppInstanceName),
			Namespace: dry.ToPtr(fakeProps.Namespace),
			Labels:    &labels,
		},
		Spec: &k8s.DeploymentSpec{
			Replicas: dry.ToPtr(float64(fakeProps.Replicas)),
			Selector: &k8s.LabelSelector{
				MatchLabels: &labels,
			},
			Template: &k8s.PodTemplateSpec{
				Metadata: &k8s.ObjectMeta{
					Labels: &labels,
				},
				Spec: &k8s.PodSpec{
					SecurityContext: &k8s.PodSecurityContext{
						RunAsNonRoot: dry.ToPtr(true),
						RunAsUser:    dry.ToPtr(float64(1000)),
						RunAsGroup:   dry.ToPtr(float64(1000)),
					},
					Containers: &[]*k8s.Container{
						{
							Name:            dry.ToPtr("fake-ea"),
							Image:           dry.ToPtr(fakeProps.Image),
							ImagePullPolicy: dry.ToPtr(fakeProps.ImagePullPolicy),
							Command:         dry.PtrSlice(fakeProps.Command),
							Args:            dry.PtrSlice(fakeProps.Args),
							Env:             &envVars,
							Ports:           convertToK8sPorts(fakeProps.Ports),
							SecurityContext: &k8s.SecurityContext{
								RunAsUser:    dry.ToPtr(float64(1000)),
								RunAsGroup:   dry.ToPtr(float64(1000)),
								RunAsNonRoot: dry.ToPtr(true),
							},
						},
					},
				},
			},
		},
	})

	// Create services for each port
	for _, port := range fakeProps.Ports {
		servicePorts := []*k8s.ServicePort{
			{
				Name:       dry.ToPtr(port.Name),
				Port:       dry.ToPtr(float64(port.ContainerPort)),
				TargetPort: k8s.IntOrString_FromNumber(dry.ToPtr(float64(port.ContainerPort))),
				Protocol:   dry.ToPtr(port.Protocol),
			},
		}

		serviceName := fmt.Sprintf("%s-%s", fakeProps.AppInstanceName, port.Name)

		_, err := servicev1.New(ctx, &servicev1.Props{
			Name:        serviceName,
			Namespace:   fakeProps.Namespace,
			AppName:     "fake",
			AppInstance: fakeProps.AppInstanceName,
			ServiceType: "ClusterIP",
			Ports:       servicePorts,
			Selector:    labels,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create service for port %s: %w", port.Name, err)
		}
	}

	return Result{
		Component: chart,
		nodeName:  fakeProps.AppInstanceName,
		namespace: fakeProps.Namespace,
	}, nil
}
