package cmd

import (
	"fmt"
	"github.com/jedib0t/go-pretty/table"
	"github.com/jedib0t/go-pretty/text"
	"github.com/mallardduck/ob-charts-tool/internal/charts"
	"github.com/spf13/cobra"
	"io"
	"os"
)

// verifyChartImagesCmd represents the verifyChartImages command
var verifyChartImagesCmd = &cobra.Command{
	Use:   "verifyChartImages",
	Short: "Verify that the rancher mirrored images for a target monitoring chart exist",
	Long: `Using either a version as first arg, or helm chart debug output from STDIN, this command will output a list
of the necessary images used in the chart. And then verify those are mirrored by the Rancher Image mirror.`,
	Args: func(cmd *cobra.Command, args []string) error {
		// Check if there's one argument provided
		if len(args) == 1 {
			return nil
		}

		// Check if there is data coming from stdin
		if isDataFromStdin() {
			return nil
		}

		return fmt.Errorf("you must provide either one argument or input from stdin")
	},
	Run: verifyChartImagesHandler,
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

// Helper function to determine if there's data from stdin
func isDataFromStdin() bool {
	info, err := os.Stdin.Stat()
	if err != nil {
		return false
	}

	// Check if stdin is not a terminal and there is data to read
	return info.Mode()&os.ModeCharDevice == 0
}

func verifyChartImagesHandler(cmd *cobra.Command, args []string) {
	if len(args) == 1 {
		fmt.Println("Received argument:", args[0])
		// TODO: fetch chart and process it, then do STDIN route
	} else if isDataFromStdin() {
		// Read stdin data
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			fmt.Println("Error reading stdin:", err)
			return
		}
		fmt.Println(text.AlignCenter.Apply("Starting to process from stdin...", 75))
		imagesLists, err := charts.PrepareChartImagesList(string(data))
		err = charts.ProcessRenderedChartImages(&imagesLists)
		if err != nil {
			return
		}
		checkedImages := charts.CheckRancherImages(imagesLists.RancherImages)
		t := table.NewWriter()
		t.SetOutputMirror(os.Stdout)
		t.AppendHeader(table.Row{"#", "Image", "Status"})
		idx := 0
		for image, status := range checkedImages {
			idx++
			statusIcon := "✅"
			if !status {
				statusIcon = "❌"
			}
			t.AppendRow(table.Row{
				idx,
				image,
				statusIcon,
			})
		}
		t.Render()
	}
}
