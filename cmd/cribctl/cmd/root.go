package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/smartcontractkit/crib-sdk/internal/core/common/dry"
)

var cfgFile string

var cfgDirFn = sync.OnceValues(func() (string, error) {
	home, err := os.UserHomeDir()
	cfgDir := filepath.Join(home, ".cribctl")
	return cfgDir, dry.FirstError(err, os.MkdirAll(cfgDir, 0o700))
})

// rootCmd represents the base command when called without any subcommands.
var RootCmd = &cobra.Command{
	Use:   "cribctl",
	Short: "A brief description of your application",
	Long: `A longer description that spans multiple lines and likely contains
examples and usage of using your application. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		bindFlags(cmd)
	},
	// Uncomment the following line if your bare application
	// has an action associated with it:
	// Run: func(cmd *cobra.Command, args []string) { },
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute(ctx context.Context) {
	if err := RootCmd.ExecuteContext(ctx); err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	RootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.cribctl.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	RootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Search config in home directory with name ".cribctl" (without extension).
		viper.AddConfigPath(configDirectory())
		viper.SetConfigType("yaml")
		viper.SetConfigName(".cribctl")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}
}

func configDirectory() string {
	home, err := cfgDirFn()
	cobra.CheckErr(err)
	return home
}

// bindFlags binds flags available to Cobra commands to Viper.
func bindFlags(cmd *cobra.Command) {
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		// Bind only if the flag hasn't been set by a parent command
		if !f.Changed && viper.IsSet(f.Name) {
			val := viper.Get(f.Name)
			if err := cmd.Flags().Set(f.Name, fmt.Sprint(val)); err != nil {
				// TODO: Use a logger instead of fmt.
				_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Unable to set flag %s: %v\n", f.Name, err)
			}
		}

		// Bind the flag to viper.
		if err := viper.BindPFlag(f.Name, f); err != nil {
			// TODO: Use a logger instead of fmt.
			_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Unable to bind flag %s: %v\n", f.Name, err)
		}
	})

	// Bind local flags if they exist.
	cmd.LocalFlags().VisitAll(func(f *pflag.Flag) {
		if err := viper.BindPFlag(f.Name, f); err != nil {
			// TODO: Use a logger instead of fmt.
			_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Unable to bind local flag %s: %v\n", f.Name, err)
		}
	})
}
