package cmd

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var (
	// Version represents the current version of the chart build scripts
	Version = "v0.0.0-dev"
	// GitCommit represents the latest commit when building this script
	GitCommit = "HEAD"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "ob-charts-tool",
	Short: "A brief description of your application",
	Long: `A longer description that spans multiple lines and likely contains
examples and usage of using your application. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Version: fmt.Sprintf("v%s (%s)", Version, GitCommit),
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	logLevel := viper.GetString("log_level")
	if logLevel == "" {
		logLevel = "warn"
	}

	// Set up logging based on logLevel
	switch strings.ToLower(logLevel) {
	case "debug":
		log.SetOutput(os.Stderr)
		log.SetLevel(log.DebugLevel)
	case "info":
		log.SetOutput(os.Stderr)
		log.SetLevel(log.InfoLevel)
	case "warn":
		log.SetOutput(os.Stderr)
		log.SetLevel(log.WarnLevel)
	case "error":
		log.SetOutput(os.Stderr)
		log.SetLevel(log.ErrorLevel)
	default:
		log.Println("Invalid log level")
	}

	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	viper.SetEnvPrefix("OB")
	viper.AutomaticEnv()
	err := viper.BindEnv("log_level")
	if err != nil {
		return
	}
}
