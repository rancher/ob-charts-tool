package syncprom

import (
	"encoding/json"
	"fmt"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/rancher/ob-charts-tool/internal/cmd/updatemonitoringmixins/jsonnet"
	"github.com/rancher/ob-charts-tool/internal/cmd/updatemonitoringmixins/types"
	mainGit "github.com/rancher/ob-charts-tool/internal/git"
	"github.com/sirupsen/logrus"
	"os"
	"path/filepath"
)

func prepareGitRules(currentState *chartState, tempDir string, chart types.RulesGitSource, chartPath string) error {
	if chart.Source == "" {
		chart.Source = "_mixin.jsonnet"
	}

	url := chart.Repository.RepoURL
	baseName := filepath.Base(url)
	clonePath := filepath.Join(tempDir, baseName)

	// Remove the clonePath if it exists from previous runs...
	_ = os.RemoveAll(clonePath)

	branch := "main"
	if chart.Repository.Branch != "" {
		branch = chart.Repository.Branch
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

	logrus.Infof("Cloning %s to %s", chart.Repository.RepoURL, clonePath)
	_, cloneErr := mainGit.CachedShallowCloneRepository(configParams, clonePath)
	if cloneErr != nil {
		return cloneErr
	}

	if chart.Mixin != "" {
		currentState.cwd = tempDir

		sourceCwd := chart.Cwd
		mixinFile := chart.Source

		mixinDir := filepath.Join(clonePath, sourceCwd)
		currentState.mixinDir = mixinDir
		jbErr := jsonnet.InitJsonnetBuilder(mixinDir)
		if jbErr != nil {
			return jbErr
		}

		// TODO this is where python checks for content field in charts

		logrus.Infof("Generatring rules from %s", mixinFile)
		vm := jsonnet.NewVm(currentState.mixinDir)
		renderedJson, err := vm.EvaluateAnonymousSnippet(
			mixinFile,
			chart.Mixin,
		)
		if err != nil {
			return err
		}

		var alerts Alerts
		jsonErr := json.Unmarshal([]byte(renderedJson), &alerts)
		if jsonErr != nil {
			return jsonErr
		}
		currentState.alerts = alerts
		currentState.url = url
	} else {
		sourcePath := filepath.Join(tempDir, chart.Source)
		sourceContent, err := os.ReadFile(sourcePath)
		if err != nil {
			return err
		}
		currentState.rawText = string(sourceContent)
		fmt.Println(currentState.rawText)
		// TODO parse to alerts
		currentState.alerts = Alerts{}
	}

	return nil
}
