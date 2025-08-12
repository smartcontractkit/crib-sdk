package service

import (
	"context"
	"fmt"
	"reflect"

	"github.com/aws/constructs-go/constructs/v10"
	"github.com/cdk8s-team/cdk8s-core-go/cdk8s/v2"

	"github.com/smartcontractkit/crib-sdk/internal"
	"github.com/smartcontractkit/crib-sdk/internal/core/common/infra"
)

type (
	// IChartFactory provides a clean interface for components to create charts without boilerplate.
	// The framework automatically provides an implementation that handles context and resource ID generation.
	//
	// Example usage in a component:
	//   func (c *MyComponent) Apply(factory IChartFactory) *MyResult {
	//       chart := factory.NewChart(c)
	//       // Use chart to create Kubernetes resources...
	//       return &MyResult{Chart: chart}
	//   }
	//
	// This eliminates the need for these repetitive lines in every component:
	//   parent := internal.ConstructFromContext(ctx)
	//   chart := cdk8s.NewChart(parent, crib.ResourceID("MyComponent", props), nil)
	IChartFactory interface {
		// NewChart creates and returns a new cdk8s.Chart. The name and values of the chart are derived
		// from the provided value `v`, which should implement fmt.Stringer for a meaningful name.
		NewChart(v any, opts ...ChartOptFn) cdk8s.Chart
	}

	// ChartFactory implements IChartFactory and is automatically injected by the framework.
	ChartFactory struct {
		ctxFn func() context.Context
	}

	chartOpts struct {
		parent     constructs.Construct // The parent construct for the chart, typically derived from the context.
		resourceID *string              // Optional resource ID for the chart, if not provided, it will be generated from the value's type name.
		chartProps *cdk8s.ChartProps    // Optional properties for the chart.
	}

	ChartOptFn func(*chartOpts)
)

// NewChartFactory returns a new instance of ChartFactory, setting an initial context which
// may be overridden later by the Apply method.
func NewChartFactory(ctx context.Context) func() *ChartFactory {
	return func() *ChartFactory {
		return &ChartFactory{
			ctxFn: func() context.Context {
				return ctx
			},
		}
	}
}

// WithParent sets or overrides the parent construct for the chart.
func WithParent(parent constructs.Construct) ChartOptFn {
	return func(opts *chartOpts) {
		opts.parent = parent
	}
}

// WithResourceID sets or overrides the resource ID for the chart.
func WithResourceID(resourceID *string) ChartOptFn {
	return func(opts *chartOpts) {
		opts.resourceID = resourceID
	}
}

// WithChartProps sets or overrides the properties for the chart.
func WithChartProps(props *cdk8s.ChartProps) ChartOptFn {
	return func(opts *chartOpts) {
		opts.chartProps = props
	}
}

func defaultResourceID(v any) *string {
	name := func() string {
		if v, ok := v.(fmt.Stringer); ok {
			return v.String()
		}
		if v, ok := v.(interface{ Name() string }); ok {
			return v.Name()
		}
		// If the value does not implement fmt.Stringer, use its type name as a fallback.
		return reflect.TypeOf(v).Name()
	}()
	return infra.ResourceID(name, v)
}

// NewChart implements ChartFactory and creates a new cdk8s.Chart.
// It handles the boilerplate of getting the parent construct, generating resource IDs, and creating the chart.
func (c *ChartFactory) NewChart(v any, opts ...ChartOptFn) cdk8s.Chart {
	opts = append(
		[]ChartOptFn{
			WithParent(internal.ConstructFromContext(c.ctxFn())),
			WithResourceID(defaultResourceID(v)),
			WithChartProps(nil),
		},
		opts...,
	)
	chartOpts := new(chartOpts)
	for _, opt := range opts {
		opt(chartOpts)
	}
	return cdk8s.NewChart(chartOpts.parent, chartOpts.resourceID, chartOpts.chartProps)
}

// Apply implements the component interface for chartFactory, making it injectable.
func (c *ChartFactory) Apply(ctx context.Context) IChartFactory {
	c.ctxFn = func() context.Context { return ctx }
	return c
}

func (c *ChartFactory) String() string {
	return "sdk.composite.builtin.ChartFactory"
}
