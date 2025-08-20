package cmd

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	helpers "github.com/rancher/ob-charts-tool/internal/cmd"
	"github.com/rancher/ob-charts-tool/internal/cmd/chartimages"
	"github.com/rancher/ob-charts-tool/internal/logging"

	"github.com/jedib0t/go-pretty/table"
	"github.com/jedib0t/go-pretty/text"
	"github.com/spf13/cobra"
)

// verifyChartImagesCmd represents the verifyChartImages command
var verifyChartImagesCmd = &cobra.Command{
	Use:   "verifyChartImages",
	Short: "Verify that the rancher mirrored images for a target monitoring chart exist",
	Long: `Using either a version as first arg, or helm chart debug output from STDIN, this command will output a list
of the necessary images used in the chart. And then verify those are mirrored by the Rancher Image mirror.`,
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
	Run: verifyChartImagesHandler,
}

func init() {
	rootCmd.AddCommand(verifyChartImagesCmd)
}

func verifyChartImagesHandler(_ *cobra.Command, args []string) {
	var data []byte
	var err error
	if len(args) == 1 {
		targetVersion := args[0]
		fmt.Println(
			text.AlignCenter.Apply(
				text.Color.Sprintf(text.FgBlue, "Looking for `rancher-monitoring` chart with version `%s`...", targetVersion),
				125,
			),
		)
		cwd, err := os.Getwd()
		if err != nil {
			panic(err)
		}

		chartTargetRoot := fmt.Sprintf("%s/charts/rancher-monitoring/%s", cwd, targetVersion)
		if _, err := os.Stat(chartTargetRoot); os.IsNotExist(err) {
			panic(fmt.Sprintf("Cannot find a monitoring chart with the provided version (%s)", targetVersion))
		}

		fmt.Println(
			text.Color.Sprintf(text.FgBlue, "The %s chart was found - next this tool will run `helm debug` to get the rendered chart.", targetVersion),
		)

		var helmArgs string
		if _, err := os.Stat(fmt.Sprintf("%s/debug.yaml", cwd)); !os.IsNotExist(err) {
			helmArgs = fmt.Sprintf("template --debug rancher-monitoring %s -f %s/debug.yaml -n cattle-monitoring-system", chartTargetRoot, cwd)
		} else {
			helmArgs = fmt.Sprintf("template --debug rancher-monitoring %s -n cattle-monitoring-system", chartTargetRoot)
		}
		logging.Log.Debug(helmArgs)
		cmd := exec.Command("helm", strings.Split(helmArgs, " ")...)
		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		err = cmd.Run()
		if err != nil {
			logging.Log.Error(stderr.String())
			panic(err)
		}

		data = stdout.Bytes()

	} else if helpers.IsDataFromStdin() {
		// Read stdin data
		data, err = io.ReadAll(os.Stdin)
		if err != nil {
			fmt.Println("Error reading stdin:", err)
			return
		}
		fmt.Println(text.AlignCenter.Apply(text.Color.Sprint(text.FgBlue, "Starting to process from stdin..."), 75))
	}
	if len(data) > 0 {
		processHelmChartImages(string(data))
	}
}

func processHelmChartImages(helmChart string) {
	imagesLists := chartimages.PrepareChartImagesList(helmChart)
	err := chartimages.ProcessRenderedChartImages(&imagesLists)
	if err != nil {
		return
	}
	checkedImages := chartimages.CheckRancherImages(imagesLists.RancherImages)
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
