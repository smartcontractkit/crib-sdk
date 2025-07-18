package internal

import (
	"errors"
	"fmt"
	"io/fs"

	"gopkg.in/yaml.v3"
)

// ChartRef represents a default reference to a chart.
// Scalar components can define default chart references that can be compiled
// and interpreted by the chart loader.
type (
	ChartRef struct {
		Values map[string]any `yaml:"values" validate:"required,dive,omitempty"`
		Chart  Chart          `yaml:"chart"  validate:"required"`
	}

	Chart struct {
		Name        string `yaml:"name"        validate:"required,lte=63,dns_rfc1035_label"`
		ReleaseName string `yaml:"releaseName" validate:"required,lte=63,dns_rfc1035_label"`
		Repository  string `yaml:"repository"  validate:"required,startswith=http|startswith=oci"`
		Version     string `yaml:"version"     validate:"required,lte=63,semver|eq=main"`
	}
)

func (c *ChartRef) Validate() error {
	v, err := NewValidator()
	if err != nil {
		return err
	}
	return v.Struct(c)
}

// NewChartRef parses the referenced chart from the given file handler.
func NewChartRef(s fs.FS, path string) (ref *ChartRef, err error) {
	f, err := s.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %s: %w", path, err)
	}
	defer func() {
		err = errors.Join(err, f.Close())
	}()

	ref = new(ChartRef)
	if err := yaml.NewDecoder(f).Decode(ref); err != nil {
		return nil, fmt.Errorf("failed to decode chart reference: %w", err)
	}
	if err := ref.Validate(); err != nil {
		return nil, fmt.Errorf("failed to validate chart reference: %w", err)
	}
	return ref, nil
}
