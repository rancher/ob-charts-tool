package verifysubchartimages

import (
	"fmt"
	"github.com/go-cmd/cmd"
	"github.com/rancher/ob-charts-tool/internal/cmd/verifysubchartimages/static"
	"github.com/rancher/ob-charts-tool/internal/fs"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
	"strings"
)

const (
	AppVersionHelmKey          = ".Chart.AppVersion"
	DefaultUseAppVersionStanza = "default " + AppVersionHelmKey
)

func VerifySubchartImages(workingPath, targetVersion, packageTargetRoot string) {
	logrus.Infof("%s is clean, proceeding to `make prepare` the package", workingPath)
	// Run make prepare to get the chart built fresh
	err := runPrepareCommand(workingPath, packageTargetRoot)
	if err != nil {
		logrus.Fatal(err)
	}

	logrus.Info("Will verify each subchart based on pre-defined rules...")
	err = static.ProcessCharts(packageTargetRoot)
	if err != nil {
		logrus.Fatal(err)
	}

	logrus.Info("Preparing patch now...")

	// Run make patch now to finish patching
	err = runPatchCommandGroup(workingPath, packageTargetRoot)
	if err != nil {
		logrus.Fatal(err)
	}

	logrus.Info("Cleaning git repo now...")

	// Run make patch now to finish patching
	err = runCleanCommand(workingPath, packageTargetRoot)
	if err != nil {
		logrus.Fatal(err)
	}
}

// DynamicVerifySubchartImages works like VerifySubchartImages but attempts to use chart analysis to identify where to update charts
// This is extremely EXPERIMENTAL and should NOT be used as a primary tool yet; meaning Devs can use it but the Dev is responsible for results.
func DynamicVerifySubchartImages(workingPath, targetVersion, packageTargetRoot string) {
	logrus.Infof("%s is clean, proceeding to `make prepare` the package", workingPath)
	// Run make prepare to get the chart built fresh
	err := runPrepareCommand(workingPath, packageTargetRoot)
	if err != nil {
		logrus.Fatal(err)
	}

	// This area is a WIP and works dynamically to find references...
	// Find charts that refer to `.Chart.AppVersion` and use it with `default`
	// As well as identifying where/what that is being used as a default for to inform what value tags to update
	chartsPath := filepath.Join(packageTargetRoot)
	subCharts, searchErr := fs.FindSubdirsWithStringInFile(chartsPath, AppVersionHelmKey)
	if searchErr != nil {
		logrus.Fatal(searchErr)
	}
	logrus.Infof("found %d subcharts that use %s with defaults", len(subCharts), AppVersionHelmKey)
	logrus.Info(subCharts)

	// Collect a list of the metadata for Charts.yaml we need from each subchart
	chartMeta := collectChartMeta(chartsPath, subCharts)
	logrus.Infof("found %d subcharts", len(chartMeta))
	logrus.Info(chartMeta)

	//  This is where we identify what values are connected to the usages of `.Chart.AppVersion`
	chartMetaWithRefs := collectAppVersionRefs(chartsPath, chartMeta)
	logrus.Infof("found %d subcharts", len(chartMetaWithRefs))
	logrus.Info(chartMetaWithRefs)

	// Verify each subchart & warn about outdated value then update
	logrus.Infof("do more")
	// TODO: actually implement updates for dynamic mode...
	logrus.Fatal("This is not fully implemented yet, sorry.")
}

func collectChartMeta(rootPath string, subCharts []string) []ChartMetadata {
	var chartMetas []ChartMetadata
	for _, subChart := range subCharts {
		chartYamlPath := filepath.Join(rootPath, subChart, "charts", "Chart.yaml")
		yamlBytes, err := os.ReadFile(chartYamlPath)
		if err != nil {
			logrus.Fatal(err)
		}
		chartMeta := ChartMetadata{
			Dir: filepath.Join(rootPath, subChart),
		}
		err = yaml.Unmarshal(yamlBytes, &chartMeta)
		if err != nil {
			logrus.Fatal(err)
		}
		chartMetas = append(chartMetas, chartMeta)
	}
	return chartMetas
}

func collectAppVersionRefs(rootPath string, subCharts []ChartMetadata) []VerifyChartData {
	var appVersionRefs []VerifyChartData
	for _, subChart := range subCharts {
		references, err := fs.FindReferencesIn(subChart.Dir, AppVersionHelmKey)
		if err != nil {
			logrus.Fatal(err)
		}
		fileListMap := make(map[string]interface{})
		var fileList []string
		if len(references) > 0 {
			for _, ref := range references {
				_, has := fileListMap[ref.File]
				if !has {
					fileListMap[ref.File] = nil
				}
			}
			if len(fileListMap) > 0 {
				for name, _ := range fileListMap {
					fileList = append(fileList, name)
				}
			}
		}

		newData := VerifyChartData{
			subChart,
			fileList,
			references,
		}
		logrus.Info(newData)
	}

	return appVersionRefs
}

func runPrepareCommand(workingPath, packageTargetRoot string) error {
	chartsToolPath := filepath.Join(workingPath, "bin", "charts-build-scripts")

	parts := strings.Split(packageTargetRoot, "packages/")
	if len(parts) < 2 {
		return fmt.Errorf("packageTargetRoot must contain 'packages/'")
	}
	packageTarget := parts[1]

	// Create the command
	cmdOptions := cmd.Options{
		Buffered:  false,
		Streaming: true,
	}
	chartsCommand := cmd.NewCmdOptions(cmdOptions, chartsToolPath, "prepare", "--useCache")

	// Prepare the commands Env vars
	osEnv := os.Environ()
	chartsCommand.Env = append(osEnv, "PACKAGE="+packageTarget)

	// Print STDOUT and STDERR lines streaming from Cmd
	doneChan := make(chan struct{})
	go func() {
		defer close(doneChan)
		for chartsCommand.Stdout != nil || chartsCommand.Stderr != nil {
			select {
			case line, open := <-chartsCommand.Stdout:
				if !open {
					chartsCommand.Stdout = nil
					continue
				}
				fmt.Println(line)
			case line, open := <-chartsCommand.Stderr:
				if !open {
					chartsCommand.Stderr = nil
					continue
				}
				_, _ = fmt.Fprintln(os.Stderr, line)
			}
		}
	}()

	// Run and wait for Cmd to return, discard Status
	<-chartsCommand.Start()

	// Wait for goroutine to print everything
	<-doneChan

	logrus.Println("Chart Prepare finished successfully.")
	return nil
}

func runPatchCommandGroup(workingPath, packageTargetRoot string) error {
	for _, subChart := range static.MonitoringSubChartsWithPrefix() {
		fullPackageTarget := fmt.Sprintf("%s/%s", packageTargetRoot, subChart.String())
		err := runPatchCommand(workingPath, fullPackageTarget)
		if err != nil {
			logrus.Fatal(err)
			return err
		}
	}

	return nil
}

func runPatchCommand(workingPath, packageTargetRoot string) error {
	chartsToolPath := filepath.Join(workingPath, "bin", "charts-build-scripts")

	parts := strings.Split(packageTargetRoot, "packages/")
	if len(parts) < 2 {
		return fmt.Errorf("packageTargetRoot must contain 'packages/'")
	}
	packageTarget := parts[1]

	// TODO: iterate over MonitoringSubChartsWithPrefix() to call patch on each subchart

	// Create the command
	cmdOptions := cmd.Options{
		Buffered:  false,
		Streaming: true,
	}
	chartsCommand := cmd.NewCmdOptions(cmdOptions, chartsToolPath, "patch", "--useCache")

	// Prepare the commands Env vars
	osEnv := os.Environ()
	chartsCommand.Env = append(osEnv, "PACKAGE="+packageTarget)

	// Print STDOUT and STDERR lines streaming from Cmd
	doneChan := make(chan struct{})
	go func() {
		defer close(doneChan)
		for chartsCommand.Stdout != nil || chartsCommand.Stderr != nil {
			select {
			case line, open := <-chartsCommand.Stdout:
				if !open {
					chartsCommand.Stdout = nil
					continue
				}
				fmt.Println(line)
			case line, open := <-chartsCommand.Stderr:
				if !open {
					chartsCommand.Stderr = nil
					continue
				}
				_, _ = fmt.Fprintln(os.Stderr, line)
			}
		}
	}()

	// Run and wait for Cmd to return, discard Status
	<-chartsCommand.Start()

	// Wait for goroutine to print everything
	<-doneChan

	logrus.Println("Chart Patch command finished successfully.")

	return nil
}

func runCleanCommand(workingPath, packageTargetRoot string) error {
	chartsToolPath := filepath.Join(workingPath, "bin", "charts-build-scripts")

	parts := strings.Split(packageTargetRoot, "packages/")
	if len(parts) < 2 {
		return fmt.Errorf("packageTargetRoot must contain 'packages/'")
	}
	packageTarget := parts[1]

	// Create the command
	cmdOptions := cmd.Options{
		Buffered:  false,
		Streaming: true,
	}
	chartsCommand := cmd.NewCmdOptions(cmdOptions, chartsToolPath, "clean")

	// Prepare the commands Env vars
	osEnv := os.Environ()
	chartsCommand.Env = append(osEnv, "PACKAGE="+packageTarget)

	// Print STDOUT and STDERR lines streaming from Cmd
	doneChan := make(chan struct{})
	go func() {
		defer close(doneChan)
		for chartsCommand.Stdout != nil || chartsCommand.Stderr != nil {
			select {
			case line, open := <-chartsCommand.Stdout:
				if !open {
					chartsCommand.Stdout = nil
					continue
				}
				fmt.Println(line)
			case line, open := <-chartsCommand.Stderr:
				if !open {
					chartsCommand.Stderr = nil
					continue
				}
				_, _ = fmt.Fprintln(os.Stderr, line)
			}
		}
	}()

	// Run and wait for Cmd to return, discard Status
	<-chartsCommand.Start()

	// Wait for goroutine to print everything
	<-doneChan

	logrus.Println("Charts clean finished successfully.")

	return nil
}
