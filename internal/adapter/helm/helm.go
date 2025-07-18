package helm

import (
	"context"
	"fmt"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/smartcontractkit/crib-sdk/internal/adapter/clientsideapply"
	"github.com/smartcontractkit/crib-sdk/internal/adapter/filehandler"
	"github.com/smartcontractkit/crib-sdk/internal/core/common/dry"
	"github.com/smartcontractkit/crib-sdk/internal/core/domain"
	"github.com/smartcontractkit/crib-sdk/internal/core/port"
)

//go:generate asdf exec go tool gowrap gen -p github.com/smartcontractkit/crib-sdk/internal/core/port -i HelmClient -t timeout -g -o helm_timeout_gen.go
//go:generate asdf exec go tool gowrap gen -p github.com/smartcontractkit/crib-sdk/internal/core/port -i HelmClient -t retry -g -o helm_retry_gen.go

// Client is a Helm client that implements the port.HelmClient interface.
type Client struct {
	executor port.ClientSideApplyRunner
}

// NewClient initializes a new Helm client with the default executor.
func NewClient(_ context.Context) (port.HelmClient, error) {
	r, err := clientsideapply.NewHelmRunner()
	if err != nil {
		return nil, err
	}

	base := &Client{executor: r}
	timeoutClient := NewHelmClientWithTimeout(base, HelmClientWithTimeoutConfig{
		AddRepoTimeout:        time.Second * 30,
		CurrentVersionTimeout: time.Second * 30,
		LatestVersionTimeout:  time.Second * 30,
		ListVersionsTimeout:   time.Second * 30,
		PullRepoTimeout:       time.Second * 30,
		UpdateRepoTimeout:     time.Second * 30,
		VendorRepoTimeout:     time.Second * 90,
	})
	retryClient := NewHelmClientWithRetry(timeoutClient, 5, time.Second*5)
	return retryClient, nil
}

// VendorRepo retrieves a Helm chart from a vendor repository.
func (c *Client) VendorRepo(ctx context.Context, release port.ChartReleaser) (port.FileReader, error) {
	// Update metadata for the Helm repository if it's not an OCI repository.
	if !release.IsOCI() {
		err := dry.FirstErrorFns(
			func() error {
				return dry.Wrapf(c.AddRepo(ctx, release), "adding Helm repo %q", release)
			},
			func() error {
				return dry.Wrapf(c.UpdateRepo(ctx, release), "updating Helm repo %q", release)
			},
		)
		if err != nil {
			return nil, err
		}
	}

	return c.PullRepo(ctx, release)
}

// AddRepo invokes a `helm repo add` command to add a new Helm repository.
// Note: This method only works for http repositories, not OCI repositories.
func (c *Client) AddRepo(ctx context.Context, release port.ChartReleaser) error {
	if release.IsOCI() {
		return fmt.Errorf("cannot helm add an OCI repository: %s", release)
	}

	_, err := c.runCommand(ctx, "repo", "add", release.ChartName(), release.RepositoryURL())
	return err
}

// UpdateRepo invokes a `helm repo update` command for the specified repository.
// Note: This method only works for http repositories, not OCI repositories.
func (c *Client) UpdateRepo(ctx context.Context, release port.ChartReleaser) error {
	if release.IsOCI() {
		return fmt.Errorf("cannot helm update an OCI repository %q", release)
	}

	_, err := c.runCommand(ctx, []string{"repo", "update", release.ChartName()}...)
	return err
}

// PullRepo invokes a `helm pull` command to download a Helm chart from the specified repository.
func (c *Client) PullRepo(ctx context.Context, release port.ChartReleaser) (port.FileReader, error) {
	fh, err := filehandler.NewTempHandler(ctx, release.String())
	if err != nil {
		return nil, fmt.Errorf("creating temporary file handler for %q: %w", release, err)
	}

	// If the version is not specified or is set to the latest version, determine the latest version of the chart.
	version := release.ChartVersion().Version
	if version == domain.HelmChartLatestVersion || version == "" {
		v, err := c.LatestVersion(ctx, release)
		if err != nil {
			return nil, fmt.Errorf("getting latest version for %q: %w", release, err)
		}
		version = v.Version
	}

	args := []string{
		"pull",
		release.PullRef(),
		"--untar",
		"--untardir", fh.Name(),
		"--version", version,
	}
	if _, err := c.runCommand(ctx, args...); err != nil {
		return nil, fmt.Errorf("pulling chart %q: %w", release, err)
	}

	// Create a new FileReader for the chart directory.
	fh, err = filehandler.New(ctx, fh.AbsPathFor(dry.As[*Release](release).Name))
	return dry.Wrapf2(fh, err, "pulling chart %q", release)
}

// TemplateRepo invokes a `helm template` command to render a Helm chart into Kubernetes manifests.
func (c *Client) TemplateRepo(ctx context.Context, release port.ChartReleaser, reader port.FileReader) ([]byte, error) {
	var chart Chart
	if err := chart.Unmarshal(ctx, reader); err != nil {
		return nil, fmt.Errorf("unmarshaling chart: %w", err)
	}
	if chart.Type != domain.HelmChartTypeApplication {
		return nil, domain.ErrHelmCannotTemplate
	}

	args := []string{
		"template", release.ChartName(),
		reader.Name(),
		"-f", reader.AbsPathFor(domain.HelmValuesFileName),
	}
	res, err := c.runCommand(ctx, args...)
	return dry.Wrapf2(res.Output, err, "templating chart %q", release)
}

// ListVersions retrieves the available versions of a Helm chart from the specified repository.
func (c *Client) ListVersions(ctx context.Context, release port.ChartReleaser) (domain.HelmChartVersions, error) {
	return c.searchRepo(ctx, release, true)
}

// CurrentVersion retrieves the current locally available version of a Helm chart from the specified repository.
func (c *Client) CurrentVersion(ctx context.Context, release port.ChartReleaser) (domain.HelmChartVersion, error) {
	versions, err := c.searchRepo(ctx, release, false)
	if err != nil {
		return dry.Empty[domain.HelmChartVersion](), err
	}
	if len(versions) == 0 {
		return dry.Empty[domain.HelmChartVersion](), fmt.Errorf("no versions found for %q", release)
	}
	return versions[0], nil
}

// LatestVersion retrieves the latest version of a Helm chart from the specified repository.
func (c *Client) LatestVersion(ctx context.Context, release port.ChartReleaser) (domain.HelmChartVersion, error) {
	versions, err := c.searchRepo(ctx, release, true)
	if err != nil {
		return dry.Empty[domain.HelmChartVersion](), err
	}
	return versions.Latest(), nil
}

func (c *Client) searchRepo(ctx context.Context, release port.ChartReleaser, allVersions bool) (domain.HelmChartVersions, error) {
	args := []string{"search", "repo", release.String(), "--output", "yaml"}
	if allVersions {
		args = append(args, "--versions")
	}
	res, err := c.runCommand(ctx, args...)
	if err != nil {
		return nil, err
	}

	var versions domain.HelmChartVersions
	if err := yaml.Unmarshal(res.Output, &versions); err != nil {
		return nil, fmt.Errorf("unmarshaling search results for %q: %w", release, err)
	}

	// Loop over the version entries, deleting any that don't exactly match the search key.
	// This is because `helm search repo` is a fuzzy search. ie foo/nginx matches foo/nginx and foo/nginx-bar.
	for i := len(versions) - 1; i >= 0; i-- {
		if versions[i].Name != release.String() {
			versions = append(versions[:i], versions[i+1:]...)
		}
	}
	return versions, nil
}

func (c *Client) runCommand(ctx context.Context, args ...string) (*domain.RunnerResult, error) {
	input := &domain.ClientSideApplyManifest{
		Spec: domain.ClientSideApplySpec{
			Action: domain.ActionHelm,
			Args:   args,
		},
	}
	return c.executor.Execute(ctx, input)
}
