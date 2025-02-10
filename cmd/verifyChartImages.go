package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// verifyChartImagesCmd represents the verifyChartImages command
var verifyChartImagesCmd = &cobra.Command{
	Use:   "verifyChartImages",
	Short: "Verify that the rancher mirrored images for a target monitoring chart exist",
	Long: `Using either a version as first arg, or helm chart debug output from STDIN, this command will output a list
of the necessary images used in the chart. And then verify those are mirrored by the Rancher Image mirror.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("verifyChartImages called")
	},
}

func init() {
	rootCmd.AddCommand(verifyChartImagesCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// verifyChartImagesCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// verifyChartImagesCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
