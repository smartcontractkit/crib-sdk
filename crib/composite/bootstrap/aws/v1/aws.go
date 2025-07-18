package awsv1

import (
	"context"

	"github.com/smartcontractkit/crib-sdk/crib"
	"github.com/smartcontractkit/crib-sdk/internal"
	"github.com/smartcontractkit/crib-sdk/internal/core/common/dry"

	clientsideapply "github.com/smartcontractkit/crib-sdk/crib/scalar/clientsideapply/v1"
)

type Props struct {
	// Profile is the AWS SSO profile to use for login
	Profile string `validate:"required"`
	// Cluster is the Kubernetes cluster context to switch to
	Cluster string `validate:"required"`
	// Namespace is the Kubernetes namespace to switch to
	Namespace string `validate:"required"`
}

// Validate validates the props.
func (p *Props) Validate(ctx context.Context) error {
	return internal.ValidatorFromContext(ctx).Struct(p)
}

// Component returns a new AWS bootstrap composite component.
func Component(props *Props) crib.ComponentFunc {
	return func(ctx context.Context) (crib.Component, error) {
		if err := props.Validate(ctx); err != nil {
			return nil, err
		}
		return awsComposite(ctx, props)
	}
}

// awsComposite creates and returns a new AWS bootstrap composite component.
// This component performs three operations in sequence:
// 1. AWS SSO login with the specified profile
// 2. Switch to the specified Kubernetes cluster context
// 3. Switch to the specified Kubernetes namespace.
func awsComposite(ctx context.Context, props crib.Props) (crib.Component, error) {
	awsProps := dry.MustAs[*Props](props)

	// Step 1: AWS SSO login
	awsLogin, err := clientsideapply.New(ctx, &clientsideapply.Props{
		OnFailure: "abort",
		Action:    "aws",
		Args: []string{
			"sso",
			"login",
			"--profile",
			awsProps.Profile,
		},
	})
	if err != nil {
		return nil, err
	}

	// Step 2: Switch to cluster context
	clusterContext, err := clientsideapply.New(ctx, &clientsideapply.Props{
		OnFailure: "abort",
		Action:    "kubectx",
		Args: []string{
			awsProps.Cluster,
		},
	})
	if err != nil {
		return nil, err
	}

	// Step 3: Switch to namespace
	namespaceContext, err := clientsideapply.New(ctx, &clientsideapply.Props{
		OnFailure: "abort",
		Action:    "kubens",
		Args: []string{
			awsProps.Namespace,
		},
	})
	if err != nil {
		return nil, err
	}

	// Set up dependencies: aws login -> cluster context -> namespace
	clusterContext.Node().AddDependency(crib.ComponentState[*clientsideapply.Result](awsLogin).Component)
	namespaceContext.Node().AddDependency(crib.ComponentState[*clientsideapply.Result](clusterContext).Component)

	return namespaceContext, nil
}
