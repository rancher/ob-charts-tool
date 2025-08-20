package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/rancher/ob-charts-tool/internal/cmd/branchhints"
	"github.com/spf13/cobra"

	helpers "github.com/rancher/ob-charts-tool/internal/cmd"
	log "github.com/rancher/ob-charts-tool/internal/logging"
)

// branchQaHintCmd represents the branchQaHint command
var branchQaHintCmd = &cobra.Command{
	Use:   "branchQaHint",
	Short: "Provide the QA testing template for the active branch",
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
	Run: branchQaHintHandler,
}

func init() {
	rootCmd.AddCommand(branchQaHintCmd)
}

func branchQaHintHandler(_ *cobra.Command, args []string) {
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

	var branchQaHints string
	branchQaHints, err = branchhints.PrepareBranchHints(repoPath)
	if err != nil {
		log.Log.Fatal(err)
	}

	// Add yaml markdown braces
	branchQaHints = "```yaml\n" + branchQaHints + "```"
	fmt.Println(branchQaHints)
}
