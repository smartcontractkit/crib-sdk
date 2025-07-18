package domain

import (
	"errors"

	"github.com/smartcontractkit/crib-sdk/internal/core/common/dry"
)

// Common Helm chart types and constants.
const (
	HelmChartTypeApplication = "application"
	HelmChartTypeLibrary     = "library"
	HelmChartLatestVersion   = "latest"
	HelmChartFileName        = "Chart.yaml"          // Name of the Helm chart file.
	HelmValuesFileName       = "values.yaml"         // Name of the Helm values file.
	HelmDefaultsFileName     = "chart.defaults.yaml" // Name of the Helm defaults file.
)

var ErrHelmCannotTemplate = errors.New("cannot template helm chart, only application charts are supported")

type (
	// HelmChartVersion represents a specific version of a Helm chart.
	HelmChartVersion struct {
		Name    string `yaml:"name"`
		Version string `yaml:"version"`
	}

	// HelmChartVersions represents a collection of Helm chart versions.
	HelmChartVersions []HelmChartVersion
)

// Latest retrieves the latest version of a Helm chart from the specified repository.
func (v HelmChartVersions) Latest() HelmChartVersion {
	if len(v) == 0 {
		return dry.Empty[HelmChartVersion]()
	}
	return v[0]
}

// IsValid checks if the provided Version exists in the HelmChartVersions slice.
func (v HelmChartVersions) IsValid(version string) bool {
	for _, entry := range v {
		if entry.Version == version {
			return true
		}
	}
	return false
}
