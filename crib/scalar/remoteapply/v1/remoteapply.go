// Package remoteapplyv1 uses the [cdk8s.NewInclude] method to include a manifest from
// a remote URL. This is similar to something like `kubectl apply -f <url>` or using
// kustomize with a remote target. Through the SDK we can expose methods similar
// to what one might expect with Kustomize and being able to manipulate fields
// in the included manifest.
package remoteapplyv1

import (
	"context"

	"github.com/cdk8s-team/cdk8s-core-go/cdk8s/v2"

	"github.com/smartcontractkit/crib-sdk/crib"
	"github.com/smartcontractkit/crib-sdk/internal"
	"github.com/smartcontractkit/crib-sdk/internal/core/common/dry"
)

type Props struct {
	URL string `validate:"required,url"`
}

func (p *Props) Validate(ctx context.Context) error {
	return internal.ValidatorFromContext(ctx).Struct(p)
}

// New creates a new remote apply component. It fetches the manifest defined in the props
// and returns a component that can be used to manipulate the manifest.
func New(ctx context.Context, props crib.Props) (crib.Component, error) {
	if err := props.Validate(ctx); err != nil {
		return nil, err
	}

	chartProps := dry.MustAs[*Props](props)
	parent := internal.ConstructFromContext(ctx)
	chart := cdk8s.NewChart(parent, crib.ResourceID("sdk.RemoteApply", props), nil)

	r := cdk8s.NewInclude(chart, crib.ResourceID("Default", props), &cdk8s.IncludeProps{
		Url: dry.ToPtr(chartProps.URL),
	})
	chart.Node().SetDefaultChild(r)
	return chart, nil
}
