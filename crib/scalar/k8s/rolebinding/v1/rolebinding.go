package v1

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
	Namespace   string         `validate:"required"`
	AppName     string         `validate:"required"`
	AppInstance string         `validate:"required"`
	Name        string         `validate:"required"`
	RoleRef     *k8s.RoleRef   `validate:"required"`
	Subjects    []*k8s.Subject `validate:"required,gt=0,dive,required"`
}

type RoleBinding struct {
	crib.Component
	props Props
}

func (r *RoleBinding) Name() string {
	return r.props.Name
}

func (p *Props) Validate(ctx context.Context) error {
	return internal.ValidatorFromContext(ctx).Struct(p)
}

// New creates a new Kubernetes RoleBinding scalar component.
func New(ctx context.Context, props crib.Props) (*RoleBinding, error) {
	if err := props.Validate(ctx); err != nil {
		return nil, err
	}

	scalarProps := dry.MustAs[*Props](props)
	parent := internal.ConstructFromContext(ctx)
	chart := cdk8s.NewChart(parent, crib.ResourceID("sdk.scalar.RoleBindingV1", props), nil)

	resourceID := crib.ResourceID(domain.CDK8sResource, props)

	resourceMetadataProps := &domain.DefaultResourceMetadataProps{
		Namespace:    scalarProps.Namespace,
		AppName:      scalarProps.AppName,
		AppInstance:  scalarProps.AppInstance,
		ResourceName: scalarProps.Name,
	}
	metadataFactory, err := domain.NewMetadataFactory(resourceMetadataProps)
	if err != nil {
		return nil, dry.Wrapf(err, "failed to create default metadata for rolebinding")
	}
	metadata := metadataFactory.K8sResourceMetadata()

	roleBinding := k8s.NewKubeRoleBinding(chart, resourceID, &k8s.KubeRoleBindingProps{
		Metadata: metadata,
		RoleRef:  scalarProps.RoleRef,
		Subjects: dry.ToPtr(scalarProps.Subjects),
	})

	scalar := &RoleBinding{
		Component: roleBinding,
		props:     *scalarProps,
	}

	return scalar, nil
}
