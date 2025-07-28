package crib

import (
	"context"

	"github.com/smartcontractkit/crib-sdk/internal/core/common/dry"
	"github.com/smartcontractkit/crib-sdk/internal/core/service"
)

// NewComposite returns a ComponentFunc that creates a composite from the given scalars.
// All scalars within a single Composite share the same Composite context. The Composite API
// provides a dependency graph allowing Scalars to produce and consume results that are
// shared within the graph. The ordering of Scalar Components within the Composite does not
// matter as the graph of producers vs consumers is calculated and Scalars are automatically
// executed based on when they are available to the graph.
//
// Scalars within a Composite can be repeated multiple times. If that Scalar produces a consumable
// result, the result will be automatically be collected as a slice of that result-type and provided
// to the consumer.
//
// To adopt the Composite API, a Scalar Component should include a Constructor that returns a concrete type.
// The recommendation is that this concrete type include any props that the Scalar needs to operate. The operates
// under the assumption that all Scalars are stateless and can exist in isolation.
// Additionally, the Scalar Component needs to have a method named `Apply`, similar to implementing an interface.
// The difference is that the `Apply` method can accept any number of parameters and return any number of results which
// are then automatically collected and passed to the next Scalar in the Composite graph.
//
// Within the framework, there are a few built-in types that are automatically provided
// to any requesting Scalar component. These are:
//
//	// context.Context
//	func (MyComponent) Apply(ctx context.Context) {
//		// ctx is an instance of the parent context passed while applying a Plan and includes the root CDK8s Chart object.
//	}
//
//	// service.ChartFactory
//	func (m MyComponent) Apply(factory service.ChartFactory) {
//		// factory provides some lightweight helper methods for creating charts and alleviates some
//		// boilerplate of getting the parent construct, generating resource IDs, and creating the chart.
//		// Example:
//		// chart := factory.CreateChart("my-chart", m)
//	}
//
// Example:
//
//	NewComposite(
//		NewProducer("scalar1"),
//		NewProducer("scalar2"),
//		NewGroupConsumer,
//	)
//
// Caveats and limitations:
// - At the moment, getting results _out_ of a Composite is not supported, but coming soon.
func NewComposite(scalars ...any) ComponentFunc {
	cs := service.NewCompositeSet()
	return func(ctx context.Context) (Component, error) {
		component, err := cs.Apply(ctx, scalars...)
		return dry.Wrap2(component, err)
	}
}
