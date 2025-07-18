package cmd

import (
	"github.com/spf13/cobra"

	"github.com/smartcontractkit/crib-sdk/internal/adapter/cribctl"
)

// DoctorCmd represents the doctor command.
var DoctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Run a short diagnostic check to determine if your environment is ready to use cribctl.",
	Run: func(cmd *cobra.Command, _ []string) {
		ctx := cmd.Context()
		cribctl.DoctorCommand(ctx)
	},
}

func init() {
	RootCmd.AddCommand(DoctorCmd)
}
