package syncgrafana

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/sirupsen/logrus"

	"github.com/rancher/ob-charts-tool/internal/cmd/updatemonitoringmixins/jsonnet"
	"github.com/rancher/ob-charts-tool/internal/cmd/updatemonitoringmixins/types"
	mainGit "github.com/rancher/ob-charts-tool/internal/git"
	"github.com/rancher/ob-charts-tool/internal/util"
)

func prepareGitDashboard(currentState *chartState, tempDir string, chart types.DashboardGitSource, chartPath string) error {
	if chart.Source == "" {
		chart.Source = "_mixin.jsonnet"
	}

	url := chart.Repository.RepoURL
	baseName := filepath.Base(url)
	clonePath := filepath.Join(tempDir, baseName)

	// Remove the clonePath if it exists from previous runs...
	_ = os.RemoveAll(clonePath)

	branch := "main"
	if chart.Branch != "" {
		branch = chart.Branch
	}
	branchHead := chart.Repository.HeadSha
	if branchHead == "" {
		var headErr error
		branchHead, headErr = mainGit.FindBranchHeadSha(chart.Repository.RepoURL, branch)
		if headErr != nil {
			return headErr
		}
	}

	configParams := mainGit.RepoConfigParams{
		Name:   chart.Repository.Name,
		URL:    chart.Repository.RepoURL,
		Branch: branch,
		Head:   plumbing.NewHash(branchHead),
	}
	logrus.Infof("Cloning %s to %s", chart.Source, clonePath)
	_, cloneErr := mainGit.CachedShallowCloneRepository(configParams, clonePath)
	if cloneErr != nil {
		return cloneErr
	}

	mixinFile := chart.Source
	mixinDir := fmt.Sprintf("%s/%s/", clonePath, chart.Cwd)
	currentState.mixinDir = mixinDir
	jbErr := jsonnet.InitJsonnetBuilder(mixinDir)
	if jbErr != nil {
		return jbErr
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
			logrus.Errorf("Error writing to file %s: %v\n", filePath, err)
			return err
		}
	}

	mixinVarsJSON, err := json.Marshal(chart.MixinVars)
	if err != nil {
		logrus.Errorf("Error encoding mixin_vars to JSON: %v\n", err)
		return err
	}

	currentState.url = url
	currentState.cwd = tempDir
	currentState.rawText = fmt.Sprintf("((import \"%s\") + %s)", mixinFile, mixinVarsJSON)
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
	currentState.source = chart.Source
	currentState.url = chart.Source

	return nil
}

func prepareFileDashboard(currentState *chartState, chart types.DashboardFileSource, chartPath string) error {
	fileSourcePath, err := filepath.Abs(chartPath + chart.Source)
	file, err := os.ReadFile(fileSourcePath)
	if err != nil {
		return err
	}
	logrus.Infof("Generating dashboards from %s", fileSourcePath)

	currentState.rawText = string(file)
	currentState.source = chart.Source
	// TODO update to relative path
	currentState.url = chart.Source
	relPath, err := util.GetRelativePath(
		chart.GetDestination(),
		fileSourcePath,
	)
	if err == nil {
		currentState.url = relPath
	}
	return nil
}
