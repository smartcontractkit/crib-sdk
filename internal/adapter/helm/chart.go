package helm

import (
	"context"
	"errors"
	"fmt"

	"gopkg.in/yaml.v3"

	"github.com/smartcontractkit/crib-sdk/internal"
	"github.com/smartcontractkit/crib-sdk/internal/core/domain"
	"github.com/smartcontractkit/crib-sdk/internal/core/port"
)

// Chart represents the minimal structure of a Helm chart as provided by a Chart.yaml.
type Chart struct {
	Type string `default:"application" yaml:"type,omitempty" validate:"omitempty,oneof=application library"` // Type of the chart, e.g., "application", "library", etc.
}

// Validate checks if the Chart has a valid type.
func (c *Chart) Validate(ctx context.Context) error {
	return internal.ValidatorFromContext(ctx).Struct(c)
}

// Unmarshal reads the Chart.yaml file from the provided FileReader and unmarshals it into the Chart struct.
//
// Usage:
//
//	var chart *Chart
//	err := chart.Unmarshal(ctx, fileReader)
func (c *Chart) Unmarshal(ctx context.Context, r port.FileReader) (err error) {
	f, err := r.Open(domain.HelmChartFileName)
	if err != nil {
		return fmt.Errorf("opening chart file %s: %w", domain.HelmChartFileName, err)
	}
	defer func() {
		err = errors.Join(err, f.Close())
	}()

	if err := yaml.NewDecoder(f).Decode(&c); err != nil {
		return fmt.Errorf("decoding chart file %s: %w", domain.HelmChartFileName, err)
	}
	if c.Type == "" {
		c.Type = domain.HelmChartTypeApplication // Default to "application" if not specified.
	}
	return c.Validate(ctx)
}
