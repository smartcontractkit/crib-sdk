package cmd

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

var (
	// These variables are set by GoReleaser during the build process.
	version = "dev"     // will be set to the actual version during build
	commit  = "unknown" // will be set to the git commit hash during build
	date    = "unknown" // will be set to the build date during build
)

// versionCmd represents the version command.
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version information",
	Long: `Print the version information for cribctl.

This command displays the version, commit hash, build date, 
and runtime information for the cribctl binary.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Fprintf(cmd.ErrOrStderr(), "cribctl version %s\n", version)
		fmt.Fprintf(cmd.ErrOrStderr(), "  commit: %s\n", commit)
		fmt.Fprintf(cmd.ErrOrStderr(), "  built: %s\n", date)
		fmt.Fprintf(cmd.ErrOrStderr(), "  go version: %s\n", runtime.Version())
		fmt.Fprintf(cmd.ErrOrStderr(), "  platform: %s/%s\n", runtime.GOOS, runtime.GOARCH)
	},
}

func init() {
	RootCmd.AddCommand(versionCmd)
}
