package deploymentv1

import (
	"context"

	"github.com/aws/jsii-runtime-go"
	"github.com/cdk8s-team/cdk8s-core-go/cdk8s/v2"
	"github.com/cdk8s-team/cdk8s-plus-go/cdk8splus30/v2/k8s"

	"github.com/smartcontractkit/crib-sdk/crib"
	"github.com/smartcontractkit/crib-sdk/internal"
	"github.com/smartcontractkit/crib-sdk/internal/core/common/dry"
	"github.com/smartcontractkit/crib-sdk/internal/core/domain"

	workloadv1 "github.com/smartcontractkit/crib-sdk/crib/composite/workload/v1"
	ingressv1 "github.com/smartcontractkit/crib-sdk/crib/scalar/k8s/ingress/v1"
	servicev1 "github.com/smartcontractkit/crib-sdk/crib/scalar/k8s/service/v1"
)

// Props defines the configuration for creating a Kubernetes Deployment.
type Props struct {
	Name        string `validate:"required"`
	AppName     string `validate:"required"`
	AppInstance string `validate:"required"`
	Namespace   string `validate:"required"`
	// Containers is a list of container configurations to include in the Deployment pods.
	Containers []*domain.Container `validate:"dive"`
	// Volumes is a list of volumes to make available to the Deployment pods.
	Volumes    []*domain.Volume `validate:"dive"`
	PodFsGroup int
}

// DeploymentComposite represents a Kubernetes DeploymentComposite component.
// It embeds the crib.Component interface and wraps the kplus.DeploymentComposite object
// to provide additional functionality for managing Kubernetes Deployments.
type DeploymentComposite struct {
	// crib.Component holds the reference to the component cdk8plus chart
	crib.Component
	props *Props

	ctx            context.Context
	selectorLabels *map[string]*string
}

func (d *DeploymentComposite) GetAppName() string {
	return d.props.AppName
}

func (d *DeploymentComposite) GetResourceType() string {
	return "deployment"
}

// GetContainers returns the containers defined in the deployment.
func (d *DeploymentComposite) GetContainers() []*domain.Container {
	return d.props.Containers
}

// GetLabelsSelector returns the label selector for the deployment.
func (d *DeploymentComposite) GetLabelsSelector() map[string]*string {
	return dry.FromPtr(d.selectorLabels)
}

// Validate ensures that the Props struct satisfies the crib.Props interface.
func (p *Props) Validate(ctx context.Context) error {
	return internal.ValidatorFromContext(ctx).Struct(p)
}

// Component returns a new DeploymentComposite composite component.
func Component(props *Props) crib.ComponentFunc {
	return func(ctx context.Context) (crib.Component, error) {
		if err := props.Validate(ctx); err != nil {
			return nil, err
		}
		return deployment(ctx, props)
	}
}

// deployment creates a new kubernetes DeploymentComposite component. The resulting [crib.Component] represents a full intent to
// install a single DeploymentComposite resource.
func deployment(ctx context.Context, props crib.Props) (*DeploymentComposite, error) {
	if err := props.Validate(ctx); err != nil {
		return nil, err
	}

	componentProps := dry.MustAs[*Props](props)
	parent := internal.ConstructFromContext(ctx)
	deploymentChart := cdk8s.NewChart(parent, crib.ResourceID("sdk.DeploymentV1", props), nil)

	ctx = internal.ContextWithConstruct(ctx, deploymentChart)

	resourceID := crib.ResourceID(domain.CDK8sResource, props)

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

	// set defaults for deployment
	securityContext := &k8s.PodSecurityContext{
		RunAsNonRoot: dry.ToPtr(true),
	}

	if componentProps.PodFsGroup != 0 {
		securityContext.FsGroup = dry.ToPtr(float64(componentProps.PodFsGroup))
	}

	selectorLabels := metadataFactory.SelectorLabels()
	deploymentProps := &k8s.KubeDeploymentProps{
		Metadata: metadata,
		Spec: &k8s.DeploymentSpec{
			Selector: &k8s.LabelSelector{
				MatchLabels: selectorLabels,
			},
			Replicas: jsii.Number(1),
			Template: &k8s.PodTemplateSpec{
				Metadata: &k8s.ObjectMeta{
					Labels: &map[string]*string{
						"app.kubernetes.io/name":     dry.ToPtr(componentProps.AppName),
						"app.kubernetes.io/instance": dry.ToPtr(componentProps.AppInstance),
					},
				},
				Spec: &k8s.PodSpec{
					Containers:      domain.ConvertContainers(componentProps.Containers),
					Volumes:         domain.ConvertVolumes(componentProps.Volumes),
					SecurityContext: securityContext,
				},
			},
		},
	}

	k8s.NewKubeDeployment(deploymentChart, resourceID, deploymentProps)

	i := &DeploymentComposite{
		ctx:            ctx,
		Component:      deploymentChart,
		props:          componentProps,
		selectorLabels: selectorLabels,
	}

	return i, nil
}

// GetNamespace returns the namespace of the deployment.
func (d *DeploymentComposite) GetNamespace() string {
	return d.props.Namespace
}

// GetAppInstance returns the application instance identifier.
func (d *DeploymentComposite) GetAppInstance() string {
	return d.props.AppInstance
}

// GetName returns the name of the deployment.
func (d *DeploymentComposite) GetName() string {
	return d.props.Name
}

// GetComponent returns the crib.Component of the deployment.
func (d *DeploymentComposite) GetComponent() crib.Component {
	return d.Component
}

func (d *DeploymentComposite) ExposeViaService(options *workloadv1.ExposeViaServiceProps) (*servicev1.Service, error) {
	return workloadv1.ExposeViaService(d.ctx, d, options)
}

func (d *DeploymentComposite) ExposeViaIngress(path string, options *workloadv1.ExposeViaIngressProps) (*ingressv1.Ingress, error) {
	return workloadv1.ExposeViaIngress(d.ctx, d, path, options)
}

// WaitForRollout waits for the Deployment rollout to complete.
// It blocks until the Deployment is fully rolled out or until the context is canceled.
// Returns an error if the rollout fails or is interrupted.
func (d *DeploymentComposite) WaitForRollout(ctx context.Context) error {
	return workloadv1.WaitForRollout(ctx, d)
}
