package secretv1

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
	StringData map[string]*string `validate:"omitempty,dive,required"`
	Immutable  *bool              `validate:"omitempty"`
	Name       string             `validate:"required"`
	Namespace  string             `validate:"required"`
	Type       string             `default:"Opaque"                   validate:"required"`
}

type Secret struct {
	crib.Component

	props Props
}

func (s *Secret) Name() string {
	return s.props.Name
}

// Validate ensures that the Props struct satisfies the crib.Props interface.
func (p *Props) Validate(ctx context.Context) error {
	return internal.ValidatorFromContext(ctx).Struct(p)
}

// New creates a new kubernetes secret component. The resulting [crib.Component] represents a full intent to
// install a single secret resource.
func New(ctx context.Context, props crib.Props) (crib.Component, error) {
	if err := props.Validate(ctx); err != nil {
		return nil, err
	}

	scalarProps := dry.MustAs[*Props](props)
	parent := internal.ConstructFromContext(ctx)
	kplusChart := cdk8s.NewChart(parent, crib.ResourceID("sdk.SecretV1", props), nil)

	resourceID := crib.ResourceID(domain.CDK8sResource, props)

	secret := k8s.NewKubeSecret(kplusChart, resourceID, &k8s.KubeSecretProps{
		Metadata: &k8s.ObjectMeta{
			Name:      dry.ToPtr(scalarProps.Name),
			Namespace: dry.ToPtr(scalarProps.Namespace),
		},
		Type:       dry.ToPtr(scalarProps.Type),
		StringData: dry.ToPtr(scalarProps.StringData),
		Immutable:  scalarProps.Immutable,
	})

	return &Secret{
		Component: secret,
		props:     *scalarProps,
	}, nil
}
