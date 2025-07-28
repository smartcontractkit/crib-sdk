package main

import (
	"context"

	registry "github.com/smartcontractkit/crib-sdk/crib/scalar/bootstrap/docker/v1"
	"github.com/smartcontractkit/crib-sdk/internal"
	"github.com/smartcontractkit/crib-sdk/internal/core/common/dry"
)

// This file demonstrates utilizing a v1 Scalar component as a producer.
// We use it to deploy a docker registry and return the name and port of the registry.

type (
	// DockerRegistry is a component that deploys a docker registry.
	// The DockerRegistry type can be used for Component Props.
	DockerRegistry struct {
		Port string `default:"5000"`
	}

	// DockerResults is a result of the DockerRegistry component. In our case, we are
	// interested in the name and port of the registry.
	// It will be shared to the Composite as *DockerResults, but also as HostnamePrinter.
	DockerResults struct {
		name string
		port string
	}
)

// NewDockerRegistry creates a new DockerRegistry component. The result of applying this Component
// is a locally deployed docker registry.
func NewDockerRegistry(port string) func() *DockerRegistry {
	return func() *DockerRegistry {
		return &DockerRegistry{Port: port}
	}
}

// Validate satisfies the port.Validator interface.
func (d *DockerRegistry) Validate(ctx context.Context) error {
	return internal.ValidatorFromContext(ctx).Struct(d)
}

// Apply satisfies the Composite API. Logic within this method is executed based on the anticipated
// ordering of the required and produced results.
//
// In this case, we accept a context and a ChartFactory, and return a *DockerResults. The return object will satisfy
// the HostnamePrinter interface.
func (d *DockerRegistry) Apply(ctx context.Context) (*DockerResults, error) {
	if err := d.Validate(ctx); err != nil {
		return nil, err
	}

	name := "registry-" + d.Port
	component := registry.Component(name, d.Port)
	_, err := component(ctx)
	return dry.Wrap2(&DockerResults{
		name: name,
		port: d.Port,
	}, err)
}

// String satisfies the Stringer interface and helps identify the type within the Composite DAG.
func (d *DockerResults) String() string {
	return "sdk.composite.kitchen-sink.DockerRegistry"
}

// Host satisfies the HostnamePrinter interface and returns the name of the registry.
func (d *DockerResults) Host() string {
	return d.name
}

// Port satisfies the HostnamePrinter interface and returns the port of the registry.
func (d *DockerResults) Port() string {
	return d.port
}
