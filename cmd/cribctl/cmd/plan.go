package cmd

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/smartcontractkit/crib-sdk/internal/adapter/filehandler"
)

var planFh *filehandler.Handler

// PlanCmd represents the plan command.
var PlanCmd = &cobra.Command{
	Use:   "plan",
	Short: "Manage CRIB-SDK Plans",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		var err error
		createFh := func() {
			planFh, err = filehandler.NewTempHandler(ctx, "cribctl-plan")
		}
		if viper.IsSet("render-dir") {
			createFh = func() {
				planFh, err = filehandler.New(ctx, viper.GetString("render-dir"))
			}
		}
		createFh()
		return err
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

func init() {
	RootCmd.AddCommand(PlanCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// PlanCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// PlanCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")

	// Flag to allow overriding the render directory for plan commands.
	PlanCmd.PersistentFlags().String("render-dir", "", "Directory to render manifests to - defaults to system temp directory")
}
