// Package loki is a crib-sdk scalar component for managing Loki helm charts.
package loki

import (
	"context"
	"embed"
	"maps"

	"github.com/cdk8s-team/cdk8s-core-go/cdk8s/v2"
	"github.com/samber/lo"

	"github.com/smartcontractkit/crib-sdk/crib"
	"github.com/smartcontractkit/crib-sdk/crib/scalar/helmchart/v1"
	"github.com/smartcontractkit/crib-sdk/internal"
	"github.com/smartcontractkit/crib-sdk/internal/core/common/dry"
)

const chartName = "helm-loki"

//go:embed chart.defaults.yaml
var defaults embed.FS

var chartDefaults *internal.ChartRef

func init() {
	var err error
	chartDefaults, err = internal.NewChartRef(defaults, "chart.defaults.yaml")
	if err != nil {
		panic(err)
	}
}

// New creates a new Loki helm chart scalar component. The resulting [crib.Component] represents a full intent to
// install a Loki instance. This component is a wrapper around the [helmchart.Chart] component and provides a
// simplified interface for creating a Loki chart.
func New(ctx context.Context, props crib.Props) (crib.Component, error) {
	chartProps := dry.MustAs[*helmchart.ChartProps](props)
	parent := internal.ConstructFromContext(ctx)
	chart := cdk8s.NewChart(parent, crib.ResourceID("sdk.Loki", props), nil)
	ctx = internal.ContextWithConstruct(ctx, chart)

	// Copy the values.
	values := make(map[string]any)
	maps.Copy(values, chartDefaults.Values)
	maps.Copy(values, chartProps.Values)

	return helmchart.New(ctx, &helmchart.ChartProps{
		Name:        dry.When(lo.IsNotEmpty(chartProps.Name), chartProps.Name, chartName),
		Chart:       dry.When(lo.IsNotEmpty(chartProps.Chart), chartProps.Chart, chartDefaults.Chart.Name),
		Namespace:   chartProps.Namespace, // TODO make configurable.
		ReleaseName: dry.When(lo.IsNotEmpty(chartProps.ReleaseName), chartProps.ReleaseName, chartDefaults.Chart.ReleaseName),
		Repo:        dry.When(lo.IsNotEmpty(chartProps.Repo), chartProps.Repo, chartDefaults.Chart.Repository),
		Values:      values,
		Version:     dry.When(lo.IsNotEmpty(chartProps.Version), chartProps.Version, chartDefaults.Chart.Version),
	})
}
