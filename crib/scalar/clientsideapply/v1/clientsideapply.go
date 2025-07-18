// Package clientsideapplyv1 represents a special client-side manifest that bridges
// current gaps between the client and server-side apply implementations. Eventually we
// anticipate this package to be deprecated in favor of server-side apply mechanisms such
// as a Crossplane XRD or a similar solution.
//
// Existence of manifests that depend on this package prevent the ability to natively
// run a `kubectl apply -f` as kubectl will not know how to handle the custom manifest.
// The idea is that `cribctl` will know how to handle this manifest and will be able to
// translate the intents of the manifest and perform the necessary operations.
//
// The schema of the fake manifest is as follows:
//
//	apiVersion: crib.smartcontract.com/v1alpha1
//	kind: ClientSideApply
//	spec:
//		onFailure: <action> # Oneof continue, abort
//		action: <action> # Oneof task, cribctl, cmd, kubectl
//		args: # cribctl shown below
//	   		- action
//	   		- contract
//	   		- -f values.yaml
//	   		- -w values.yaml=contract.address=/spec/contracts/0/address
//			- -w config.toml=/config/node/0
package clientsideapplyv1

import (
	"context"

	"github.com/cdk8s-team/cdk8s-core-go/cdk8s/v2"

	"github.com/smartcontractkit/crib-sdk/crib"
	"github.com/smartcontractkit/crib-sdk/internal"
	"github.com/smartcontractkit/crib-sdk/internal/core/common/dry"
	"github.com/smartcontractkit/crib-sdk/internal/core/domain"
)

const (
	apiVersion = "crib.smartcontract.com/v1alpha1"
	kind       = "ClientSideApply"
)

type (
	Props struct {
		Namespace string `validate:"omitempty,lte=63,dns_rfc1035_label"`

		// OnFailure is the action to take if the apply fails.
		OnFailure string `default:"abort" validate:"required,oneof=continue abort"`
		// Action is the action to take.
		Action string `validate:"required,oneof=cmd cribctl docker kind kubectl task"`
		// Args are the arguments to pass to the action.
		Args []string `validate:"required,dive"`
	}

	Result struct {
		crib.Component

		Args []string `json:"args,omitempty"`
	}
)

func (p *Props) Validate(ctx context.Context) error {
	return internal.ValidatorFromContext(ctx).Struct(p)
}

// Component returns a crib.ComponentFunc that creates a new ClientSideApply component.
func Component(props crib.Props) crib.ComponentFunc {
	return func(ctx context.Context) (crib.Component, error) {
		if err := props.Validate(ctx); err != nil {
			return nil, err
		}
		return New(ctx, props)
	}
}

// New creates a new ClientSideApply scalar component. This is a special, temporary (we hope), component
// that is able to execute client-side logic. This is a workaround for the current inability
// to perform server-side actions for certain operations.
func New(ctx context.Context, props crib.Props) (crib.Component, error) {
	chartProps := dry.MustAs[*Props](props)

	parent := internal.ConstructFromContext(ctx)
	chart := cdk8s.NewChart(parent, crib.ResourceID("sdk.ClientSideApply", props), nil)

	obj := cdk8s.NewApiObject(chart, crib.ResourceID(domain.CDK8sResource, props), &cdk8s.ApiObjectProps{
		ApiVersion: dry.ToPtr(apiVersion),
		Kind:       dry.ToPtr(kind),
		Metadata: &cdk8s.ApiObjectMetadata{
			Namespace: dry.ToPtr(chartProps.Namespace),
		},
	})
	obj.AddJsonPatch(cdk8s.JsonPatch_Add(dry.ToPtr("/spec"), map[string]any{
		"onFailure": chartProps.OnFailure,
		"action":    chartProps.Action,
		"args":      chartProps.Args,
	}))
	return &Result{
		Component: chart,
		Args:      append([]string{chartProps.Action}, chartProps.Args...),
	}, nil
}
