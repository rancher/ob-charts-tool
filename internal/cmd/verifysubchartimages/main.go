package verifysubchartimages

import (
	"errors"
	"fmt"
	"github.com/rancher/ob-charts-tool/internal/cmd/verifysubchartimages/static"
	"github.com/rancher/ob-charts-tool/internal/fs"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
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
}

func VerifySubchartImagesDynamic(workingPath, targetVersion, packageTargetRoot string) {
	logrus.Infof("%s is clean, proceeding to `make prepare` the package", workingPath)
	// Run make prepare to get the chart built fresh
	err := runPrepareCommand(workingPath, packageTargetRoot)
	if err != nil {
		logrus.Fatal(err)
	}
	// This area is a WIP and works dynamically to find references...
	// Find charts that refer to AppVersionHelmKey
	chartsPath := filepath.Join(packageTargetRoot)
	subCharts, searchErr := fs.FindSubdirsWithStringInFile(chartsPath, AppVersionHelmKey)
	if searchErr != nil {
		logrus.Fatal(searchErr)
	}
	logrus.Infof("found %d subcharts", len(subCharts))
	logrus.Info(subCharts)
	// Collect all the metadata for Charts.yaml we need from each subchart
	chartMeta := collectChartMeta(chartsPath, subCharts)
	logrus.Infof("found %d subcharts", len(chartMeta))
	logrus.Info(chartMeta)
	chartMetaWithRefs := collectAppVersionRefs(chartsPath, chartMeta)
	logrus.Infof("found %d subcharts", len(chartMetaWithRefs))
	logrus.Info(chartMetaWithRefs)
	// Verify each subchart (these may need rules coded for each chart)
	logrus.Infof("do more")
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
	chartsCommand := exec.Command(chartsToolPath, "prepare", "--useCache")

	// Set up a process group for interruptibility
	// This allows sending signals to the process and its children
	chartsCommand.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	// Setup the commands Env vars
	osEnv := os.Environ()
	chartsCommand.Env = append(osEnv, "PACKAGE="+packageTarget)

	// Create pipes for stdout/stderr to capture
	stdoutPipe, err := chartsCommand.StdoutPipe()
	if err != nil {
		return fmt.Errorf("error creating stdout pipe: %w", err)
	}
	stderrPipe, err := chartsCommand.StderrPipe()
	if err != nil {
		return fmt.Errorf("error creating stderr pipe: %w", err)
	}

	// Start the command
	if err := chartsCommand.Start(); err != nil {
		return fmt.Errorf("error starting command: %w", err)
	}

	// Create a channel to listen for OS signals
	signals := make(chan os.Signal, 1)
	// Notify the signals channel on receiving SIGINT or SIGTERM
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)

	// Goroutine to handle signals
	go func() {
		<-signals // Wait for a signal
		logrus.Println("Received termination signal, attempting to kill process group...")
		// Send SIGTERM to the process group
		// The negative PID sends the signal to the entire process group
		if chartsCommand.Process != nil {
			// Use a non-blocking kill in case the process has already exited
			err := syscall.Kill(-chartsCommand.Process.Pid, syscall.SIGTERM)
			if err != nil && !errors.Is(err, syscall.ESRCH) { // ESRCH means no such process
				logrus.Printf("Error sending SIGTERM to process group %d: %v", -chartsCommand.Process.Pid, err)
			}
		}
	}()

	// Goroutine to read from stdout pipe in real-time
	// We'll print stdout directly to os.Stdout in a Cobra CLI context
	go func() {
		_, err := io.Copy(os.Stdout, stdoutPipe)
		if err != nil && err != io.EOF {
			logrus.Printf("Error reading stdout: %v", err)
		}
	}()

	// Goroutine to read from stderr pipe in real-time and print to console
	// We'll print stderr directly to os.Stderr
	go func() {
		_, err := io.Copy(os.Stderr, stderrPipe)
		if err != nil && err != io.EOF {
			logrus.Printf("Error reading stderr: %v", err)
		}
	}()

	// Wait for the command to finish
	err = chartsCommand.Wait()
	if err != nil {
		// Return the error so Cobra can handle it (e.g., print to stderr and exit)
		return fmt.Errorf("command finished with error: %w", err)
	}

	logrus.Println("Command finished successfully.")
	return nil
}
