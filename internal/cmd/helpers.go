package cmd

import "os"

// Helper functions for cmds

// IsDataFromStdin helps to determine if there's data from stdin
func IsDataFromStdin() bool {
	info, err := os.Stdin.Stat()
	if err != nil {
		return false
	}

	// Check if stdin is not a terminal and there is data to read
	return info.Mode()&os.ModeCharDevice == 0
}
