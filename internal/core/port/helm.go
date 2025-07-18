package port

import (
	"context"

	"github.com/smartcontractkit/crib-sdk/internal/core/domain"
)

// ChartReleaser defines methods for retrieving chart references.
type ChartReleaser interface {
	Validator

	// IsOCI returns true if the chart is stored in an OCI repository.
	IsOCI() bool
	// ChartName returns the name of the repository as registered in Helm.
	// This matches our "releaseName" field in the Release struct.
	ChartName() string
	// ChartVersion returns the version of the Helm chart as a [domain.HelmChartVersion].
	ChartVersion() domain.HelmChartVersion
	// String returns a string reference to the Helm chart in the form of "<releaseName>/<name>".
	// e.g. "loki/component-chart".
	String() string
	// PullRef returns a string that can be used to pull the Helm chart from a repository.
	// For https repositories it returns the value of String(), for OCI repositories it returns the full URL.
	PullRef() string
	// RepositoryURL returns the URL of the Helm repository.
	RepositoryURL() string
}

// HelmClient defines the standard set of methods to use with a Helm client.
type HelmClient interface {
	// VendorRepo invokes a series of helm commands to vendor a Helm chart from a repository.
	// It returns a FileReader for the chart, which can be used to access the chart files.
	VendorRepo(ctx context.Context, release ChartReleaser) (FileReader, error)
	// AddRepo invokes a `helm repo add` command to add a new Helm repository.
	AddRepo(ctx context.Context, release ChartReleaser) error
	// UpdateRepo invokes a `helm repo update` command for the specified repository.
	UpdateRepo(ctx context.Context, release ChartReleaser) error
	// PullRepo invokes a `helm pull` command to download a Helm chart from the specified repository.
	// It vendors the chart in a temporary directory and returns a FileHandler for the chart.
	PullRepo(ctx context.Context, release ChartReleaser) (FileReader, error)
	// TemplateRepo invokes a `helm template` command to render a Helm chart into Kubernetes manifests.
	TemplateRepo(ctx context.Context, release ChartReleaser, reader FileReader) ([]byte, error)
	// ListVersions lists all available versions of a Helm chart for the specified repository and chart.
	ListVersions(ctx context.Context, release ChartReleaser) (domain.HelmChartVersions, error)
	// CurrentVersion returns the currently installed version of a Helm chart for the specified repository and chart.
	CurrentVersion(ctx context.Context, release ChartReleaser) (domain.HelmChartVersion, error)
	// LatestVersion returns the latest version of a Helm chart for the specified repository and chart.
	LatestVersion(ctx context.Context, release ChartReleaser) (domain.HelmChartVersion, error)
}
