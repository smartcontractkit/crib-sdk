package cmd

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/smartcontractkit/crib-sdk/internal/adapter/cribctl"
	"github.com/smartcontractkit/crib-sdk/internal/adapter/helm"
	"github.com/smartcontractkit/crib-sdk/internal/core/common/dry"
)

const (
	defaultBasePath      = "crib/scalar/charts"
	defaultScalarVersion = "v1"
)

var (
	errInvalidArgs      = errors.New("invalid arguments")
	createHelmComponent *cribctl.CreateHelmComponent
)

// helmCreateComponentCmd represents the `helm create-component` command.
// cribctl helm create-component <name> <release>@<url> [--version=<version>].
var helmCreateComponentCmd = &cobra.Command{
	Use:   "create-component <name> <release>@<url>",
	Short: "Create Helm Scalar Component",
	Long: `Automatically generate a CRIB-SDK Scalar Component from a Helm Chart. 
By default, the component will be created in the directory 'crib/scalar/<name>/v1'.

Positional arguments:

  - <name> should be the name of the component, which will be used to create the directory structure - it does not
    directly affect interacting with the Helm Chart.

  - <release> is the name of the Helm Chart release, which will be used to create the component.

  - <url> is the URL of the Helm Chart repository, which can be either an oci:// or https:// URL.
`,
	Example: `
# A scalar component named aptos will be added to crib/scalar/aptos/v1 
cribctl helm create-component aptos component-chart@https://charts.devspace.sh

# A scalar component named tailscale with the explicit version 1.84.0 will be added to crib/scalar/tailscale/v1
cribctl helm create-component tailscale tailscale-operator@https://pkgs.tailscale.com/helmcharts --version=1.84.0
`,
	Args: func(cmd *cobra.Command, args []string) error {
		// Take over the error handling to avoid printing usage on error.
		cmd.SilenceErrors = true
		cmd.SilenceUsage = true
		// Assume the user is wanting to see the help message if no arguments are provided.
		if len(args) == 0 {
			return errors.Join(cmd.Help(), errInvalidArgs)
		}

		if len(args) != 2 {
			if _, writeErr := fmt.Fprintf(cmd.ErrOrStderr(), "❌  create-component requires exactly two arguments: <name> and <release>@<url>\n\n"); writeErr != nil {
				return writeErr
			}
			return errInvalidArgs
		}

		// Determine which argument contains the release.
		var name, releaseStr string
		for _, arg := range args {
			if strings.Contains(arg, "@") {
				releaseStr = arg
				continue
			}
			name = arg
		}

		var (
			err error
			sb  strings.Builder
		)
		if releaseStr == "" {
			fmt.Fprintf(&sb, "  ❌  create-component requires a release in the format <release>@<url>\n")
			err = errors.Join(err, errInvalidArgs)
		}
		if name == "" {
			fmt.Fprintf(&sb, "  ❌  create-component requires a name for the component.\n")
			err = errors.Join(err, errInvalidArgs)
		}

		// Parse the release string to extract the name and URL.
		release, repo, _ := strings.Cut(releaseStr, "@")
		parsed, perr := url.Parse(repo)
		if perr != nil {
			fmt.Fprintf(&sb, "  ❌  create-component requires a valid URL of oci:// or https:// for the release repository: %s\n", repo)
			err = errors.Join(err, errInvalidArgs)
		}
		if parsed.Scheme != "oci" && parsed.Scheme != "https" {
			fmt.Fprintf(&sb, "  ❌  Only repositories beginning with oci or https are supported. Got: %q\n", repo)
			err = errors.Join(err, errInvalidArgs)
		}

		createHelmComponent.Release.ReleaseName = name
		createHelmComponent.Release.Name = release
		createHelmComponent.Release.Repository = repo

		if sb.Len() > 0 {
			if _, writeErr := fmt.Fprintf(cmd.ErrOrStderr(), "Input Errors:\n\n%s\n\n", sb.String()); writeErr != nil {
				return writeErr
			}
		}
		return dry.FirstError(err)
	},
	PreRunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceErrors = false

		hc, err := helm.NewClient(cmd.Context())
		if err != nil {
			return err
		}
		createHelmComponent.Client = hc

		base := viper.GetString("saveto")
		if base == "" {
			base = defaultBasePath
		}
		version := viper.GetString("scalar-version")
		if version == "" {
			version = defaultScalarVersion
		}
		createHelmComponent.Outdir = filepath.Join(base, createHelmComponent.Release.ReleaseName, version) + string(os.PathSeparator)
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		createHelmComponent.Release = &helm.Release{
			Name:        createHelmComponent.Release.Name,
			ReleaseName: createHelmComponent.Release.ReleaseName,
			Repository:  createHelmComponent.Release.Repository,
			Version:     viper.GetString("version"),
		}

		return createHelmComponent.Run(cmd.Context())
	},
}

func init() {
	createHelmComponent = &cribctl.CreateHelmComponent{
		Release: &helm.Release{},
	}

	HelmCmd.AddCommand(helmCreateComponentCmd)

	helmCreateComponentCmd.Flags().String("version", "latest", "The chart version to pull.")
	helmCreateComponentCmd.Flags().StringP("saveto", "o", defaultBasePath, "The output directory for the scalar component")
	helmCreateComponentCmd.Flags().String("scalar-version", defaultScalarVersion, "The version of the scalar component")
}
