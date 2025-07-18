package port

import (
	"context"

	"github.com/aws/constructs-go/constructs/v10"
	"github.com/cdk8s-team/cdk8s-core-go/cdk8s/v2"
)

type (
	// Planner represents methods on a [crib.Plan] to aid in building and applying a Plan.
	Planner interface {
		// Name returns the name of the plan. The name must be globally unique within Plans known to cribctl.
		// The name is used in the Plan Registry so that it can be applied with cribctl via
		// `cribctl plan apply <plan-name>`.
		//
		// When including other plans as dependencies, the caller can include it either by name
		// contrib.Plan("plan-name"), or by its function that returns the plan, e.g. examplev1.Plan().
		Name() string
		// Namespace returns the primary Kubernetes target namespace for the plan on the target cluster.
		// The Namespace will NOT be created if it does not exist.
		Namespace() string
		// Components returns a list of components that are part of the plan.
		Components() []ComponentFunc
		// ChildPlans returns a list of dependent plans that are part of the plan.
		ChildPlans() []Planner
		// Resolvers returns a list of resolvers that are part of the plan.
		Resolvers() []cdk8s.IResolver
	}

	// Component represents a Construct. Constructs are the basic building block of cdk8s.
	// They are the instrument that enables composition and creation of higher-level abstractions through normal
	// object-oriented classes.
	Component interface {
		constructs.Construct
	}

	// ComponentFunc is a function that takes a context and props and returns a Component.
	ComponentFunc func(ctx context.Context) (Component, error)
)
