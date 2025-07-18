package cmd

import (
	"github.com/spf13/cobra"
)

// HelmCmd represents the parent helm command.
var HelmCmd = &cobra.Command{
	Use:   "helm",
	Short: "Interact with Helm charts",
	PersistentPreRun: func(cmd *cobra.Command, _ []string) {
		cmd.SilenceUsage = true
	},
	RunE: func(cmd *cobra.Command, _ []string) error {
		return cmd.Help()
	},
}

func init() {
	RootCmd.AddCommand(HelmCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// HelmCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// HelmCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
