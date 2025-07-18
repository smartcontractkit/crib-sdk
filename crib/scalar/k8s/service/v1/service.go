package servicev1

import (
	"context"

	"github.com/cdk8s-team/cdk8s-core-go/cdk8s/v2"
	"github.com/cdk8s-team/cdk8s-plus-go/cdk8splus30/v2/k8s"

	"github.com/smartcontractkit/crib-sdk/crib"
	"github.com/smartcontractkit/crib-sdk/internal"
	"github.com/smartcontractkit/crib-sdk/internal/core/common/dry"
	"github.com/smartcontractkit/crib-sdk/internal/core/domain"
)

type Props struct {
	Namespace   string             `validate:"required"`
	AppName     string             `validate:"required"`
	AppInstance string             `validate:"required"`
	Name        string             `validate:"required"`
	Selector    map[string]*string `validate:"required,gt=0,dive,required"`
	ServiceType string             `validate:"required"`
	Ports       []*k8s.ServicePort `validate:"required,gt=0,dive,required"`
}

type Service struct {
	crib.Component
	props Props
}

func (s *Service) Ports() []*k8s.ServicePort {
	return s.props.Ports
}

func (s *Service) Name() string {
	return s.props.Name
}

// Validate ensures that the Props struct satisfies the crib.Props interface.
func (p *Props) Validate(ctx context.Context) error {
	return internal.ValidatorFromContext(ctx).Struct(p)
}

// New creates a new kubernetes service component. The resulting [crib.Component] represents a full intent to
// install a single service resource.
func New(ctx context.Context, props crib.Props) (*Service, error) {
	if err := props.Validate(ctx); err != nil {
		return nil, err
	}

	scalarProps := dry.MustAs[*Props](props)
	parent := internal.ConstructFromContext(ctx)
	chart := cdk8s.NewChart(parent, crib.ResourceID("sdk.ServiceV1", props), nil)

	resourceID := crib.ResourceID(domain.CDK8sResource, props)

	resourceMetadataProps := &domain.DefaultResourceMetadataProps{
		Namespace:    scalarProps.Namespace,
		AppName:      scalarProps.AppName,
		AppInstance:  scalarProps.AppInstance,
		ResourceName: scalarProps.Name,
	}
	metadataFactory, err := domain.NewMetadataFactory(resourceMetadataProps)
	if err != nil {
		return nil, dry.Wrapf(err, "failed to create default metadata for service")
	}
	metadata := metadataFactory.K8sResourceMetadata()

	service := k8s.NewKubeService(chart, resourceID, &k8s.KubeServiceProps{
		Metadata: metadata,
		Spec: &k8s.ServiceSpec{
			Type:     dry.ToPtr(scalarProps.ServiceType),
			Ports:    dry.ToPtr(scalarProps.Ports),
			Selector: dry.ToPtr(scalarProps.Selector),
		},
	})

	scalar := &Service{
		Component: service,
		props:     *scalarProps,
	}

	return scalar, nil
}
