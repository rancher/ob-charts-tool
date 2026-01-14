package cmd

import (
	"errors"
	"os"

	"github.com/rancher/ob-charts-tool/internal/cmd/branchverifycheck"
	"github.com/spf13/cobra"

	helpers "github.com/rancher/ob-charts-tool/internal/cmd"
	log "github.com/rancher/ob-charts-tool/internal/logging"
)

// branchVerifyCheck represents the branchVerifyCheck command
var branchVerifyCheck = &cobra.Command{
	Use:   "branchVerifyCheck",
	Short: "Verify branch state for chart consistency and sequential versioning",
	Args: func(_ *cobra.Command, args []string) error {
		// Check that there is either one or zero args
		if len(args) == 1 || len(args) == 0 {
			return nil
		}

		// Check if there is data coming from stdin
		if helpers.IsDataFromStdin() {
			return errors.New("does not accept input from stdin")
		}

		return errors.New("you must provide either 0 or 1 arguments")
	},
	Run: branchVerifyCheckHandler,
}

func init() {
	rootCmd.AddCommand(branchVerifyCheck)
	branchVerifyCheck.Flags().Bool("json", false, "Output results in JSON format")
}

func branchVerifyCheckHandler(cmd *cobra.Command, args []string) {
	var repoPath string
	var err error

	if len(args) > 0 {
		repoPath = args[0]
	} else {
		repoPath, err = os.Getwd()
		if err != nil {
			log.Log.Fatal(err)
		}
	}

	jsonOutput, _ := cmd.Flags().GetBool("json")
	result, err := branchverifycheck.VerifyBranch(repoPath, jsonOutput)
	if err != nil {
		log.Log.Fatal(err)
	}

	if !result.Success {
		os.Exit(1)
	}
}
