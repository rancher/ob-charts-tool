package cmd

import (
	"fmt"
	"github.com/rancher/ob-charts-tool/internal/logging"
	"github.com/spf13/viper"
	"os"

	"github.com/spf13/cobra"
)

const cliName = "ob-charts-tool"

var (
	// Version represents the current version of the chart build scripts
	Version = "v0.0.0-dev"
	// GitCommit represents the latest commit when building this script
	GitCommit = "HEAD"
)

var (
	cfgFile string
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   cliName,
	Short: "A tool for working with `ob-team-charts`",
	Long: `A CLI tool for working with the 'ob-team-charts' Helm chart repo.

Supports one-off tasks like inspecting chart contents (e.g., listing used images),
as well as automating chart maintenance workflows such as rebases.

Commands are either root-level (operating on multiple charts) or grouped
under a domain prefix (e.g., 'logging:', 'monitoring:') for chart-specific actions.`,
	Version: fmt.Sprintf("v%s (%s)", Version, GitCommit),
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// This function runs before any command's Run, RunE, PreRun, or PreRunE.
		// It's the ideal place to initialize Viper and set up logging.
		initConfig()
		logging.Configure(cmd) // Now configures Logrus (via logging internal pkg)
		return nil
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// Setup log-level global flag
	rootCmd.PersistentFlags().StringP("log-level", "l", "info", "Set the logging level (debug, info, warn, error, fatal, panic)")

	// Viper config
	viper.SetEnvPrefix("OB")
	viper.AutomaticEnv()
	err := viper.BindEnv("log_level", "OB_LOG_LEVEL")
	if err != nil {
		logging.Log.Error(err)
		return
	}

	// Bind the log-level flag to Viper (this also makes it available via viper.GetString)
	err = viper.BindPFlag("log-level", rootCmd.PersistentFlags().Lookup("log-level"))
	if err != nil {
		logging.Log.Error(err)
		return
	}
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		// Search config in home directory with name ".mycli" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigName("." + cliName)
	}

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		_, err = fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
		if err != nil {
			logging.Log.Error(err)
			return
		}
	}
}
