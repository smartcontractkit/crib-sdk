package cribctl

import (
	"context"
	"fmt"
	"os"

	"github.com/smartcontractkit/crib-sdk/internal"
	"github.com/smartcontractkit/crib-sdk/internal/adapter/filehandler"
	"github.com/smartcontractkit/crib-sdk/internal/adapter/helm"
	"github.com/smartcontractkit/crib-sdk/internal/core/common/dry"
	"github.com/smartcontractkit/crib-sdk/internal/core/domain"
	"github.com/smartcontractkit/crib-sdk/internal/core/port"
)

// cribctl helm create-component <name> <Release>@<url> --version=<version>

type CreateHelmComponent struct {
	_            struct{}
	Client       port.HelmClient
	vendorReader port.FileReader
	Release      *helm.Release
	values       map[string]any
	Outdir       string `validate:"required,dirpath,lte=255"`
}

func (c *CreateHelmComponent) Validate(ctx context.Context) error {
	return internal.ValidatorFromContext(ctx).Struct(c)
}

func (c *CreateHelmComponent) Run(ctx context.Context) error {
	err := dry.FirstErrorFns(
		func() error { return c.Validate(ctx) },
		c.vendorChart(ctx),
		c.checkVersion(ctx),
		c.readValues(ctx),
		c.generateComponent(ctx),
	)
	if err != nil {
		return fmt.Errorf("creating CRIB-SDK Helm Scalar Component %q: %w", c.Release.Name, err)
	}
	fmt.Fprintf(os.Stderr, "✅  Successfully created Helm component %q in %q\n", c.Release.Name, c.Outdir)
	return nil
}

func (c *CreateHelmComponent) vendorChart(ctx context.Context) func() error {
	return func() (err error) {
		fmt.Fprintf(os.Stderr, "ℹ️  Saving Helm metadata for %q...\n", c.Release)

		c.vendorReader, err = c.Client.VendorRepo(ctx, c.Release)
		return dry.Wrapf(err, "vendoring Helm Chart")
	}
}

func (c *CreateHelmComponent) checkVersion(ctx context.Context) func() error {
	return func() (err error) {
		needVersion := c.Release.Version == "" || c.Release.Version == "latest"
		if !needVersion {
			return nil // Skip if version is already known.
		}
		fmt.Fprintf(os.Stderr, "ℹ️  Finding latest published version for %q...\n", c.Release.Name)

		version, err := c.Client.LatestVersion(ctx, c.Release)
		if err != nil {
			return fmt.Errorf("getting latest version for %q: %w", c.Release, err)
		}
		c.Release.Version = version.Version
		return nil
	}
}

func (c *CreateHelmComponent) readValues(ctx context.Context) func() error {
	return func() (err error) {
		fmt.Fprintln(os.Stderr, "ℹ️  Loading Helm values...")

		nfh, err := filehandler.New(ctx, c.vendorReader.Name())
		if err != nil {
			return fmt.Errorf("creating file loader: %w", err)
		}

		loader, err := internal.NewFileLoaderFromFS(nfh, domain.HelmValuesFileName, internal.NewYAMLLoader())
		if err != nil {
			return fmt.Errorf("creating values file loader: %w", err)
		}
		c.values, err = loader.Values()
		return dry.Wrapf(err, "reading values file for %q", c.Release.Name)
	}
}

func (c *CreateHelmComponent) generateComponent(ctx context.Context) func() error {
	return func() (err error) {
		fmt.Fprintf(os.Stderr, "ℹ️  Generating new CRIB-SDK Scalar Component %q...\n", c.Release.Name)

		d := &helm.Defaults{
			Release: *c.Release,
			Values:  c.values,
		}
		gen, err := helm.NewGenerator(ctx, d, c.Outdir)
		if err != nil {
			return fmt.Errorf("initializing component generator: %w", err)
		}
		return gen.Generate(ctx)
	}
}
