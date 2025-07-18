// File: crib/composite/statefulset/v1/statefulset.go
package statefulsetv1

import (
	"context"

	"github.com/cdk8s-team/cdk8s-core-go/cdk8s/v2"
	"github.com/cdk8s-team/cdk8s-plus-go/cdk8splus30/v2/k8s"

	"github.com/smartcontractkit/crib-sdk/crib"
	"github.com/smartcontractkit/crib-sdk/internal"
	"github.com/smartcontractkit/crib-sdk/internal/core/common/dry"
	"github.com/smartcontractkit/crib-sdk/internal/core/domain"

	workloadv1 "github.com/smartcontractkit/crib-sdk/crib/composite/workload/v1"
)

// Props defines the configuration for creating a Kubernetes StatefulSet.
type Props struct {
	Name                 string `validate:"required"`
	AppName              string `validate:"required"`
	Namespace            string `validate:"required"`
	AppInstance          string `validate:"required"`
	Labels               map[string]string
	Annotations          map[string]string
	Containers           []*domain.Container             `validate:"dive"`
	Volumes              []*domain.Volume                `validate:"dive"`
	PodManagementPolicy  string                          // Parallel or OrderedReady
	VolumeClaimTemplates []*domain.PersistentVolumeClaim `validate:"dive"`
}

// StatefulSetComposite represents a Kubernetes StatefulSet component.
type StatefulSetComposite struct {
	crib.Component
	ctx             context.Context
	props           *Props
	kubeStatefulSet k8s.KubeStatefulSet
	selectorLabels  *map[string]*string
}

func (s *StatefulSetComposite) GetAppName() string {
	return s.props.AppName
}

func (s *StatefulSetComposite) GetResourceType() string {
	return "statefulset"
}

// Validate ensures that the Props struct satisfies the crib.Props interface.
func (p *Props) Validate(ctx context.Context) error {
	return internal.ValidatorFromContext(ctx).Struct(p)
}

// GetNamespace implements the WorkloadResource interface.
func (s *StatefulSetComposite) GetNamespace() string {
	return s.props.Namespace
}

// GetAppInstance implements the WorkloadResource interface.
func (s *StatefulSetComposite) GetAppInstance() string {
	return s.props.AppInstance
}

// GetName implements the WorkloadResource interface.
func (s *StatefulSetComposite) GetName() string {
	return s.props.Name
}

// GetComponent returns the crib.Component of the deployment.
func (s *StatefulSetComposite) GetComponent() crib.Component {
	return s.Component
}

// GetContainers returns the containers defined in the deployment.
func (s *StatefulSetComposite) GetContainers() []*domain.Container {
	return s.props.Containers
}

// GetLabelsSelector returns the label selector for the deployment.
func (s *StatefulSetComposite) GetLabelsSelector() map[string]*string {
	return *s.selectorLabels
}

// Component returns a new StatefulSetComposite component.
func Component(props *Props) crib.ComponentFunc {
	return func(ctx context.Context) (crib.Component, error) {
		if err := props.Validate(ctx); err != nil {
			return nil, err
		}
		return statefulset(ctx, props)
	}
}

// statefulset creates a new Kubernetes StatefulSet component.
func statefulset(ctx context.Context, props crib.Props) (*StatefulSetComposite, error) {
	if err := props.Validate(ctx); err != nil {
		return nil, err
	}

	componentProps := dry.MustAs[*Props](props)
	parent := internal.ConstructFromContext(ctx)
	statefulsetChart := cdk8s.NewChart(parent, crib.ResourceID("sdk.StatefulSetV1", props), nil)

	ctx = internal.ContextWithConstruct(ctx, statefulsetChart)

	resourceID := crib.ResourceID(domain.CDK8sResource, props)

	i := &StatefulSetComposite{
		Component: statefulsetChart,
		ctx:       ctx,
		props:     componentProps,
	}

	resourceMetadataProps := &domain.DefaultResourceMetadataProps{
		ResourceName: componentProps.Name,
		AppName:      componentProps.AppName,
		Namespace:    componentProps.Namespace,
		AppInstance:  componentProps.AppInstance,
	}
	metadataFactory, err := domain.NewMetadataFactory(resourceMetadataProps)
	if err != nil {
		return nil, dry.Wrapf(err, "failed to create default metadata for deployment")
	}
	metadata := metadataFactory.K8sResourceMetadata()

	volumeClaimTemplates, err := workloadv1.ConvertVolumeClaimTemplates(dry.MustAs[workloadv1.WorkloadResource](i), componentProps.VolumeClaimTemplates)
	if err != nil {
		return nil, dry.Wrapf(err, "failed to create default volumeclaim templates for statefulset")
	}
	i.selectorLabels = metadataFactory.SelectorLabels()
	i.kubeStatefulSet = k8s.NewKubeStatefulSet(statefulsetChart, resourceID, &k8s.KubeStatefulSetProps{
		Metadata: metadata,

		Spec: &k8s.StatefulSetSpec{
			ServiceName: dry.ToPtr(componentProps.Name),
			Replicas:    dry.ToPtr(float64(1)),
			Selector: &k8s.LabelSelector{
				MatchLabels: i.selectorLabels,
			},
			Template: &k8s.PodTemplateSpec{
				Metadata: &k8s.ObjectMeta{
					Labels: metadata.Labels,
				},
				Spec: &k8s.PodSpec{
					Containers: domain.ConvertContainers(componentProps.Containers),
					Volumes:    domain.ConvertVolumes(componentProps.Volumes),
				},
			},
			VolumeClaimTemplates: volumeClaimTemplates,
		},
	})

	return i, nil
}

// WaitForRollout waits for the StatefulSet rollout to complete.
func (s *StatefulSetComposite) WaitForRollout(ctx context.Context) error {
	return workloadv1.WaitForRollout(ctx, s)
}
