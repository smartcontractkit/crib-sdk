package kindv1

import (
	"context"
	"fmt"

	"github.com/smartcontractkit/crib-sdk/crib"
	"github.com/smartcontractkit/crib-sdk/internal"
	"github.com/smartcontractkit/crib-sdk/internal/core/common/dry"

	registry "github.com/smartcontractkit/crib-sdk/crib/scalar/bootstrap/docker/v1"
	kind "github.com/smartcontractkit/crib-sdk/crib/scalar/bootstrap/kind/v1"
	clientsideapply "github.com/smartcontractkit/crib-sdk/crib/scalar/clientsideapply/v1"
)

type Props struct {
	Name string `validate:"required"`
}

// Validate validates the props.
func (p *Props) Validate(ctx context.Context) error {
	return internal.ValidatorFromContext(ctx).Struct(p)
}

// Component returns a new kind composite component.
func Component(props *Props) crib.ComponentFunc {
	return func(ctx context.Context) (crib.Component, error) {
		if err := props.Validate(ctx); err != nil {
			return nil, err
		}
		return kindComposite(ctx, props)
	}
}

// kindComposite creates and returns a new kind composite component.
// This component creates a kind cluster and sets the context to the cluster.
func kindComposite(ctx context.Context, props crib.Props) (crib.Component, error) {
	kindProps := dry.MustAs[*Props](props)

	// Create the registry container using the registry scalar component
	dockerRegistry, err := registry.Component(fmt.Sprintf("%s-registry", kindProps.Name), "5001")(ctx)
	if err != nil {
		return nil, err
	}

	// Create the kind cluster using the kind scalar component
	cluster, err := kind.Component(kindProps.Name)(ctx)
	if err != nil {
		return nil, err
	}

	clusterContext, err := clientsideapply.New(ctx, &clientsideapply.Props{
		Namespace: "default",
		OnFailure: "abort",
		Action:    "kubectl",
		Args: []string{
			"config",
			"use-context",
			fmt.Sprintf("kind-%s", kindProps.Name),
		},
	})
	if err != nil {
		return nil, err
	}

	// Set up dependencies: docker registry -> cluster -> context
	cluster.Node().AddDependency(crib.ComponentState[*clientsideapply.Result](dockerRegistry).Component)
	clusterContext.Node().AddDependency(crib.ComponentState[*clientsideapply.Result](cluster).Component)

	return clusterContext, nil
}
