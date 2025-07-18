package namespacev1

import (
	"context"

	"github.com/cdk8s-team/cdk8s-core-go/cdk8s/v2"

	"github.com/smartcontractkit/crib-sdk/crib"
	"github.com/smartcontractkit/crib-sdk/internal"
	"github.com/smartcontractkit/crib-sdk/internal/core/common/dry"
	"github.com/smartcontractkit/crib-sdk/internal/core/domain"

	cdk8splus "github.com/cdk8s-team/cdk8s-plus-go/cdk8splus30/v2"
)

type Props struct {
	Namespace string `default:"default" validate:"omitempty,lte=63,dns_rfc1035_label"`
}

func (p *Props) Validate(ctx context.Context) error {
	return internal.ValidatorFromContext(ctx).Struct(p)
}

// Component creates and returns a new [crib.ComponentFunc] with intention to create a new Namespace.
func Component(namespace string) crib.ComponentFunc {
	props := &Props{
		Namespace: namespace,
	}
	return func(ctx context.Context) (crib.Component, error) {
		if err := props.Validate(ctx); err != nil {
			return nil, err
		}
		return New(ctx, props)
	}
}

// New creates a new namespace and returns the component.
func New(ctx context.Context, props crib.Props) (crib.Component, error) {
	chartProps := dry.MustAs[*Props](props)
	parent := internal.ConstructFromContext(ctx)
	c := cdk8s.NewChart(parent, crib.ResourceID("sdk.Namespace", props), nil)

	cdk8splus.NewNamespace(c, crib.ResourceID(domain.CDK8sResource, props), &cdk8splus.NamespaceProps{
		Metadata: &cdk8s.ApiObjectMetadata{
			Name: dry.ToPtr(chartProps.Namespace),
		},
	})
	return c, nil
}
