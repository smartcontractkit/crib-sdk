package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/smartcontractkit/crib-sdk/internal/adapter/cribctl"
)

// previewCmd represents the preview command.
var previewCmd = &cobra.Command{
	Use:   "preview",
	Short: "Preview a CRIB-SDK Plan's DAG structure",
	Long: `Preview displays the Directed Acyclic Graph (DAG) structure of a CRIB-SDK Plan
without applying it to the cluster. This is useful for debugging and understanding
the dependency relationships between components in the plan.

When the --render-dir flag is provided, the generated Kubernetes manifests will be
written to the specified directory instead of the default temporary location.`,
	Args: cribctl.ValidatePlanArgs("preview"),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Preview the plan using the unified function
		preview, outputDir, err := cribctl.PreviewPlan(cmd.Context(), planFh, args[0])
		if err != nil {
			return fmt.Errorf("previewing plan: %w", err)
		}
		if _, err := fmt.Fprintf(cmd.ErrOrStderr(), "Plan DAG Preview for %s:\n\n%s\n", args[0], preview); err != nil {
			return fmt.Errorf("writing preview output: %w", err)
		}

		if viper.IsSet("render-dir") {
			if _, err := fmt.Fprintf(cmd.ErrOrStderr(), "\nGenerated files dumped to: %s\n", outputDir); err != nil {
				return fmt.Errorf("writing output directory info: %w", err)
			}
		}
		return nil
	},
}

func init() {
	PlanCmd.AddCommand(previewCmd)
}
