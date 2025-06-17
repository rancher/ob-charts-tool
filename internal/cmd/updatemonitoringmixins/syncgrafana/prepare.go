package syncgrafana

import (
	"encoding/json"
	"fmt"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/rancher/ob-charts-tool/internal/cmd/updatemonitoringmixins/types"
	mainGit "github.com/rancher/ob-charts-tool/internal/git"
	"github.com/sirupsen/logrus"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
)

func prepareGitDashboard(currentState *chartState, tempDir string, chart types.DashboardGitSource, chartPath string) error {
	url := chart.Repository.RepoURL
	logrus.Infof("Clone %s", url)
	baseName := filepath.Base(url)
	logrus.Infof("base %s", baseName)
	clonePath := filepath.Join(tempDir, baseName)
	logrus.Infof("cp %s", clonePath)

	// Remove the clonePath if it exists from previous runs...
	_ = os.RemoveAll(clonePath)

	configParams := mainGit.RepoConfigParams{
		Name: chart.Repository.Name,
		URL:  chart.Repository.RepoURL,
		Head: plumbing.NewHash(chart.Branch),
	}
	_, err := mainGit.CachedShallowCloneRepository(configParams, clonePath)
	if err != nil {
		return err
	}

	mixinFile := chart.Source
	mixinDir := fmt.Sprintf("%s/%s/", clonePath, chart.Cwd)
	jsonnetFile := filepath.Join(mixinDir, "jsonnetfile.json")
	if _, err := os.Stat(jsonnetFile); !os.IsNotExist(err) {
		fmt.Println("Running jsonnet-bundler, because jsonnetfile.json exists")

		cmd := exec.Command("jb", "install")
		cmd.Dir = mixinDir // Set the working directory
		err := cmd.Run()
		if err != nil {
			fmt.Printf("Error running jsonnet-bundler: %v\n", err)
		}
	}

	filePath := filepath.Join(mixinDir, mixinFile)
	if chart.Content != "" {
		file, err := os.Create(filePath)
		if err != nil {
			return err
		}
		defer file.Close() // Ensure the file is closed when the function exits
		_, err = file.WriteString(chart.Content)
		if err != nil {
			fmt.Printf("Error writing to file %s: %v\n", filePath, err)
			return err
		}
	}

	mixinVarsJSON, err := json.Marshal(chart.MixinVars)
	if err != nil {
		fmt.Printf("Error encoding mixin_vars to JSON: %v\n", err)
		return err
	}

	currentState.cwd = tempDir
	currentState.rawText = fmt.Sprintf("((import \"%s\" + %s)", mixinFile, mixinVarsJSON)
	currentState.source = filepath.Base(mixinFile)
	return nil
}

func prepareUrlDashboard(currentState *chartState, chart types.DashboardURLSource) error {
	logrus.Infof("Generating dashboard from %s", chart.Source)

	resp, err := http.Get(chart.Source)
	if err != nil {
		return err
	}
	defer resp.Body.Close() // Ensure the connection is closed

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	currentState.rawText = string(body)

	return nil
}

func prepareFileDashboard(currentState *chartState, chart types.DashboardFileSource, chartPath string) error {
	fileSourcePath, err := filepath.Abs(chartPath + chart.Source)
	file, err := os.ReadFile(fileSourcePath)
	if err != nil {
		return err
	}

	currentState.rawText = string(file)

	return nil
}
