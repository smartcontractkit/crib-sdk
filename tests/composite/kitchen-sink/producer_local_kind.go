package main

import (
	"context"

	"github.com/smartcontractkit/crib-sdk/internal"
	"github.com/smartcontractkit/crib-sdk/internal/core/common/dry"

	kind "github.com/smartcontractkit/crib-sdk/crib/scalar/bootstrap/kind/v1"
)

// This file demonstrates utilizing a v1 Scalar component as a producer.
// We use it to deploy a local kind cluster and return the name of the cluster.
// This demonstrates how we can use the Composite API to "do" something totally different,
// but provide results that implement the same interface.

type (
	// KindCluster is a component that deploys a local kind cluster.
	// The KindCluster type can be used for Component Props.
	KindCluster struct {
		Name string `default:"local-kind"`
	}

	// KindResults is a result of the KindCluster component. In our case, we are
	// interested in the name of the cluster.
	// It will be shared to the Composite as *KindResults, but also as HostnamePrinter.
	KindResults struct {
		name string
	}
)

// NewKindCluster creates a new KindCluster component. The result of applying this Component
// is a locally deployed kind cluster.
func NewKindCluster() *KindCluster {
	return &KindCluster{}
}

// Validate satisfies the port.Validator interface.
func (k *KindCluster) Validate(ctx context.Context) error {
	return internal.ValidatorFromContext(ctx).Struct(k)
}

// Apply satisfies the Composite API. Logic within this method is executed based on the anticipated
// ordering of the required and produced results.
//
// In this case, we accept a context and return a *KindResults. The return object will satisfy
// the HostnamePrinter interface.
func (k *KindCluster) Apply(ctx context.Context) (*KindResults, error) {
	if err := k.Validate(ctx); err != nil {
		return nil, err
	}

	component := kind.Component(k.Name)
	_, err := component(ctx)
	return dry.Wrap2(&KindResults{
		name: k.Name,
	}, err)
}

// String satisfies the Stringer interface and helps identify the type within the Composite DAG.
func (k *KindResults) String() string {
	return "sdk.composite.kitchen-sink.KindCluster"
}

// Host satisfies the HostnamePrinter interface and returns the name of the cluster.
func (k *KindResults) Host() string {
	return k.name
}

// Port satisfies the HostnamePrinter interface and returns an empty string.
// This is because the KindCluster component does not produce a port.
func (k *KindResults) Port() string {
	return ""
}
