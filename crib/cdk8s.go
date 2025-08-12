package crib

import (
	"context"

	"github.com/cdk8s-team/cdk8s-core-go/cdk8s/v2"

	"github.com/smartcontractkit/crib-sdk/internal/core/service"
)

// NewChart provides a new cdk8s.Chart instance.
func NewChart(ctx context.Context, v any) cdk8s.Chart {
	return service.NewChartFactory(ctx)().NewChart(v)
}
