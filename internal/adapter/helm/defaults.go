package helm

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"github.com/smartcontractkit/crib-sdk/internal"
	"github.com/smartcontractkit/crib-sdk/internal/core/common/dry"
	"github.com/smartcontractkit/crib-sdk/internal/core/domain"
	"github.com/smartcontractkit/crib-sdk/internal/core/port"
)

type (
	// Defaults represents the Defaults file used by the generator to provide default values for a Helm chart.
	Defaults struct {
		Values  map[string]any `yaml:"values"            validate:"omitempty,yaml"`
		Release Release        `yaml:"chart,omitempty"   validate:"required"`
		Version string         `yaml:"version,omitempty" validate:"omitempty,version"`
	}

	// Release represents the Helm chart release information.
	Release struct {
		Name        string `yaml:"name,omitempty"        validate:"required"`                                // Name of the Helm chart.
		ReleaseName string `yaml:"releaseName,omitempty" validate:"required"`                                // Name of the Helm release.
		Repository  string `yaml:"repository,omitempty"  validate:"required,startswith=http|startswith=oci"` // Repository URL of the Helm chart.
		Version     string `yaml:"version,omitempty"     validate:"required,version"`                        // Version of the Helm chart.
	}
)

// Save writes the Defaults struct to a file named chart.defaults.yaml in the provided FileWriter.
func (d *Defaults) Save(ctx context.Context, r port.FileWriter) (err error) {
	if err = d.Validate(ctx); err != nil {
		return fmt.Errorf("validating chart defaults: %w", err)
	}

	f, err := r.Create(domain.HelmDefaultsFileName)
	if err != nil {
		return fmt.Errorf("creating chart defaults file %s: %w", domain.HelmDefaultsFileName, err)
	}
	defer func() {
		err = errors.Join(err, f.Close())
	}()

	return yaml.NewEncoder(f).Encode(d)
}

func (d *Defaults) Validate(ctx context.Context) error {
	return internal.ValidatorFromContext(ctx).Struct(d)
}

func (d *Defaults) Marshal() ([]byte, error) {
	return yaml.Marshal(d)
}

// Unmarshal reads the chart.defaults.yaml file from the provided FileReader and unmarshals it into the Defaults struct.
//
// Usage:
//
//	var d Defaults // Important, do not use a pointer here.
//	err := d.Unmarshal(ctx, fileReader)
func (d *Defaults) Unmarshal(ctx context.Context, r port.FileReader) (err error) {
	f, err := r.Open(domain.HelmDefaultsFileName)
	if err != nil {
		return fmt.Errorf("opening chart file %q: %w", domain.HelmChartFileName, err)
	}
	defer func() {
		err = errors.Join(err, f.Close())
	}()

	return dry.FirstError(
		yaml.NewDecoder(f).Decode(&d),
		d.Validate(ctx),
	)
}

// IsOCI satisfies the [port.ChartReleaser] interface, indicating whether the release is an
// OCI (Open Container Initiative) repository.
func (r *Release) IsOCI() bool {
	// Check if the repository URL starts with "oci://", indicating it's an OCI repository.
	return len(r.Repository) >= 6 && r.Repository[:6] == "oci://"
}

// ChartName satisfies the [port.ChartReleaser] interface, returning the name of the chart as registered with
// Helm.
func (r *Release) ChartName() string {
	return r.ReleaseName
}

// ChartVersion satisfies the [port.ChartReleaser] interface, returning the version of the Helm chart.
func (r *Release) ChartVersion() domain.HelmChartVersion {
	return domain.HelmChartVersion{
		Name:    r.String(),
		Version: r.Version,
	}
}

// String satisfies the [port.ChartReleaser], it returns a string reference to the Helm chart in the form
// of "<releaseName>/<name>". e.g. "loki/component-chart".
func (r *Release) String() string {
	return filepath.Join(r.ReleaseName, r.Name)
}

// PullRef satisfies the [port.ChartReleaser] interface, returning a string that can be used to pull the Helm chart.
// For OCI repositories, it returns the full URL, for HTTP repositories it returns the String().
func (r *Release) PullRef() string {
	if r.IsOCI() {
		return r.Repository // For OCI, return the full repository URL.
	}
	return r.String() // For HTTP, return the chart reference.
}

// RepositoryURL satisfies the [port.ChartReleaser] interface, returning the URL of the Helm repository.
func (r *Release) RepositoryURL() string {
	return r.Repository
}

func (r *Release) Validate(ctx context.Context) error {
	return internal.ValidatorFromContext(ctx).Struct(r)
}
