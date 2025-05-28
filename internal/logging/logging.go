package logging

import (
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"os"
)

// Log is the global Logrus logger instance for the application.
var Log *logrus.Logger

func init() {
	// Initialize the logger instance and its basic formatter/output
	// This runs once when the package is imported.
	Log = logrus.New()
	Log.SetOutput(os.Stdout)
	Log.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
		DisableColors: false, // Set to true if running in a non-color-enabled terminal
	})
	// Default level before configuration
	Log.SetLevel(logrus.InfoLevel)
}

// Configure sets up the Logrus logger's level based on Viper's configuration
// and the command's flags. It takes the *cobra.Command to inspect flag changes.
func Configure(cmd *cobra.Command) {
	// Viper reads the 'log-level' key, respecting the precedence order:
	// flag > env var > config file > default
	levelStr := viper.GetString("log-level")

	level, err := logrus.ParseLevel(levelStr)
	if err != nil {
		Log.Warnf("Unknown log level '%s'. Defaulting to 'info'.", levelStr)
		Log.SetLevel(logrus.InfoLevel)
		return
	}
	Log.SetLevel(level)

	// Determine the source of the log level for informational messages
	source := getLogLevelSource(cmd)
	Log.Debugf("Logrus level set to: %s (source: %s)", level.String(), source)
}

// getLogLevelSource determines whether the log level came from a flag,
// environment variable, config file, or default.
func getLogLevelSource(cmd *cobra.Command) string {
	// Check if the log-level flag was explicitly set on the command line.
	if flag := cmd.PersistentFlags().Lookup("log-level"); flag != nil && flag.Changed {
		return "flag"
	}
	// Check if the environment variable was set.
	if os.Getenv("OB_LOG_LEVEL") != "" { // Using the specific env var name
		return "environment variable"
	}
	// Check if it came from the config file.
	if viper.ConfigFileUsed() != "" && viper.IsSet("log-level") {
		return "config file"
	}
	return "default"
}
