package jobv1

import (
	"context"

	"github.com/cdk8s-team/cdk8s-core-go/cdk8s/v2"

	"github.com/smartcontractkit/crib-sdk/crib"
	"github.com/smartcontractkit/crib-sdk/internal"
	"github.com/smartcontractkit/crib-sdk/internal/core/common/dry"
	"github.com/smartcontractkit/crib-sdk/internal/core/domain"

	kplus "github.com/cdk8s-team/cdk8s-plus-go/cdk8splus30/v2"
)

type Props struct {
	JobProps *kplus.JobProps `validate:"required"`
}

// Validate ensures that the Props struct satisfies the crib.Props interface.
func (p *Props) Validate(ctx context.Context) error {
	return internal.ValidatorFromContext(ctx).Struct(p)
}

// New creates a new kubernetes job component. The resulting [crib.Component] represents a full intent to
// install a single job resource.
func New(ctx context.Context, props crib.Props) (crib.Component, error) {
	if err := props.Validate(ctx); err != nil {
		return nil, err
	}

	scalarProps := dry.MustAs[*Props](props)
	parent := internal.ConstructFromContext(ctx)
	kplusChart := cdk8s.NewChart(parent, crib.ResourceID("sdk.JobV1", props), nil)

	jobResourceID := crib.ResourceID(domain.CDK8sResource, props)
	job := kplus.NewJob(kplusChart, jobResourceID, scalarProps.JobProps)

	return job, nil
}
