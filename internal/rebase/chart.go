package rebase

import (
	"fmt"
	"github.com/mallardduck/ob-charts-tool/internal/upstream"
	"os"
	"strings"

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/mallardduck/ob-charts-tool/internal/git"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

func (s *StartRequest) CollectRebaseChartsInfo() ChartRebaseInfo {
	rebaseInfo := ChartRebaseInfo{
		TargetVersion:     s.TargetVersion,
		FoundChart:        s.FoundChart,
		ChartDependencies: s.ChartDependencies,
	}

	for _, item := range rebaseInfo.ChartDependencies {
		log.Debugf("Fetching chart dependencies for: %v", item)
		newestTagInfo := findNewestReleaseTagInfo(item)
		if newestTagInfo != nil {
			rebaseInfo.DependencyChartVersions = append(rebaseInfo.DependencyChartVersions, *newestTagInfo)
		}
	}

	return rebaseInfo
}

func findNewestReleaseTagInfo(chartDep ChartDep) *DependencyChartVersion {
	exists, tag := findNewestReleaseTag(chartDep)
	if !exists {
		return nil
	}
	return &DependencyChartVersion{
		Name: chartDep.Name,
		Ref:  tag.Name().String(),
		Hash: tag.Hash().String(),
	}
}

func findNewestReleaseTag(chartDep ChartDep) (bool, *plumbing.Reference) {
	version := chartDep.Version
	if strings.Contains(version, ".*") {
		version = strings.ReplaceAll(version, ".*", "")
	}

	repo := upstream.IdentifyChartUpstream(chartDep.Name)
	tag := fmt.Sprintf("%s-%s", chartDep.Name, version)

	found, tags := git.FindTagsMatching(repo, tag)
	if !found {
		panic("Could not find any tags for this chart")
	}

	highestTag := git.FindHighestVersionTag(tags, chartDep.Name)

	return found, highestTag
}

func (s *ChartRebaseInfo) FindChartsContainers() error {
	// TODO: look up main charts values file and find images from there
	fmt.Println("TODO find containers for: " + s.FoundChart.Name + "@" + s.FoundChart.CommitHash)

	for _, item := range s.DependencyChartVersions {
		// TODO: find each dependency chart's Chart.yaml and values.yaml file
		fmt.Println("TODO find containers for: " + item.Name + "@" + item.Hash)
	}
	return nil
}

func (s *ChartRebaseInfo) SaveStateToRebaseYaml(saveDir string) {
	yamlData, err := yaml.Marshal(s)
	if err != nil {
		log.Fatalf("Error marshaling YAML: %v", err)
	}

	err = os.WriteFile(fmt.Sprintf("%s/rebase.yaml", saveDir), yamlData, 0644)
	if err != nil {
		log.Fatalf("Error writing YAML to file: %v", err)
	}
}
