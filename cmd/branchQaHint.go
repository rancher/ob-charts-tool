package cmd

import (
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
		// Check if there's one argument provided
		if len(args) == 1 {
			return nil
		}

		// Check if there is data coming from stdin
		if helpers.IsDataFromStdin() {
			return nil
		}

		return fmt.Errorf("you must provide either one argument or input from stdin")
	},
	Run: branchQaHintHandler,
}

func init() {
	rootCmd.AddCommand(branchQaHintCmd)
}

func branchQaHintHandler(_ *cobra.Command, args []string) {
	var cwd string
	var err error

	if cwd, err = os.Getwd(); err != nil {
		log.Log.Fatal(err)
	}

	var branchQaHints string
	branchQaHints, err = branchhints.PrepareBranchHints(cwd)
	if err != nil {
		log.Log.Fatal(err)
	}

	// Add yaml markdown braces
	branchQaHints = "```yaml\n" + branchQaHints + "```"
	fmt.Println(branchQaHints)
}
