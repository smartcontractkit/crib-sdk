package cmd

import (
	"fmt"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/smartcontractkit/crib-sdk/internal/adapter/cribctl"
)

// applyCmd represents the apply command.
var applyCmd = &cobra.Command{
	Use:   "apply",
	Short: "Apply a CRIB-SDK Plan",
	Long: `Apply a CRIB-SDK Plan to the target cluster. 
	
The command will first show a preview of the plan's DAG structure, then prompt for confirmation before applying.`,
	Args: cribctl.ValidatePlanArgs("apply"),
	Run: func(cmd *cobra.Command, args []string) {
		planName := args[0]
		autoAccept := viper.GetBool("yes")

		// Show preview first
		if _, err := fmt.Fprintf(cmd.ErrOrStderr(), "Previewing plan %q...\n\n", planName); err != nil {
			return
		}
		preview, _, err := cribctl.PreviewPlan(cmd.Context(), planFh, planName)
		if err != nil {
			if _, err := fmt.Fprintf(cmd.ErrOrStderr(), "Error previewing plan: %v\n", err); err != nil {
				return
			}
			return
		}
		if _, err := fmt.Fprintln(cmd.ErrOrStderr(), preview); err != nil {
			return
		}

		// If auto-accept is enabled, skip confirmation
		if autoAccept {
			if _, err := fmt.Fprintln(cmd.ErrOrStderr(), "\nAuto-accepting (--yes flag provided)..."); err != nil {
				return
			}
		} else {
			// Prompt for confirmation
			var confirmed bool
			confirm := huh.NewConfirm().
				Title("Apply this plan?").
				Description(fmt.Sprintf("This will apply plan %q to the target cluster.", planName)).
				Affirmative("Yes, apply").
				Negative("No, cancel").
				Value(&confirmed)

			if err := confirm.Run(); err != nil {
				if _, err := fmt.Fprintf(cmd.ErrOrStderr(), "Error during confirmation: %v\n", err); err != nil {
					return
				}
				return
			}
			if !confirmed {
				if _, err := fmt.Fprintln(cmd.ErrOrStderr(), "Plan application cancelled."); err != nil {
					return
				}
				return
			}
		}

		// Apply the plan
		if _, err := fmt.Fprintf(cmd.ErrOrStderr(), "\nApplying plan %q...\n", planName); err != nil {
			return
		}
		if err := cribctl.ApplyPlan(cmd.Context(), planFh, planName); err != nil {
			if _, err := fmt.Fprintf(cmd.ErrOrStderr(), "Error applying plan: %v\n", err); err != nil {
				return
			}
			return
		}
		if _, err := fmt.Fprintf(cmd.ErrOrStderr(), "Successfully applied plan: %s\n", planName); err != nil {
			return
		}
	},
}

func init() {
	PlanCmd.AddCommand(applyCmd)

	// Add the -y/--yes flag for auto-accepting
	applyCmd.Flags().BoolP("yes", "y", false, "Auto-accept the confirmation prompt")

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// applyCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// applyCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
