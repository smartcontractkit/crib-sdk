package crib

import (
	"io/fs"

	"github.com/smartcontractkit/crib-sdk/internal"
)

// ChartRef represents a default reference to a chart.
// Scalar components can define default chart references that can be compiled
// and interpreted by the chart loader.
type ChartRef = internal.ChartRef

// Chart represents chart metadata.
type Chart = internal.Chart

// NewChartRef parses the referenced chart from the given file handler.
func NewChartRef(s fs.FS, path string) (ref *ChartRef, err error) {
	return internal.NewChartRef(s, path)
}
