package upstream

import (
	"fmt"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/mallardduck/ob-charts-tool/internal/git"
	"github.com/mallardduck/ob-charts-tool/internal/rebase"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
	"os"
	"strings"
)

const (
	upstreamGrafanaChartsUrl    = "https://github.com/grafana/helm-charts.git"
	upstreamPrometheusChartsUrl = "https://github.com/prometheus-community/helm-charts.git"
	upstreamVersionTemplate     = "kube-prometheus-stack-%s"
)

func ChartVersionExists(version string) (bool, string) {
	return git.VerifyTagExists(upstreamPrometheusChartsUrl, fmt.Sprintf(upstreamVersionTemplate, version))
}

func PrepareRebaseRequestInfo(version string, gitHash string) rebase.StartRequest {
	rebaseRequest := rebase.StartRequest{
		TargetVersion:    version,
		TargetCommitHash: gitHash,
	}

	rebaseRequest.FetchChart()
	rebaseRequest.FindChartDeps()

	return rebaseRequest
}

type RebaseInfo struct {
	TargetVersion           string                   `yaml:"target_version"`
	TargetCommitHash        string                   `yaml:"commit_hash"`
	ChartFileUrl            string                   `yaml:"chart_file_url"`
	ChartDependencies       []rebase.ChartDep        `yaml:"chart_dependencies"`
	DependencyChartVersions []DependencyChartVersion `yaml:"dependency_chart_versions"`
	ChartsImagesLists       map[string][]string      `yaml:"charts_images_lists"`
}

type DependencyChartVersion struct {
	Name    string `yaml:"name"`
	Version string `yaml:"version"`
	Hash    string `yaml:"hash"`
}

func CollectRebaseChartsInfo(request rebase.StartRequest) RebaseInfo {
	rebaseInfo := RebaseInfo{
		TargetVersion:     request.TargetVersion,
		TargetCommitHash:  request.TargetCommitHash,
		ChartFileUrl:      request.ChartFileUrl,
		ChartDependencies: request.ChartDependencies,
	}

	for _, item := range rebaseInfo.ChartDependencies {
		// TODO: Fetch newest charts for each dependency
		fmt.Println(item)
		newestTagInfo := findNewestReleaseTagInfo(item)
		if newestTagInfo != nil {
			rebaseInfo.DependencyChartVersions = append(rebaseInfo.DependencyChartVersions, *newestTagInfo)
		}
	}

	return rebaseInfo
}

func findNewestReleaseTag(chartDep rebase.ChartDep) (bool, *plumbing.Reference) {
	version := chartDep.Version
	if strings.Contains(version, ".*") {
		version = strings.ReplaceAll(version, ".*", "")
	}

	repo := upstreamPrometheusChartsUrl
	tag := fmt.Sprintf("%s-%s", chartDep.Name, version)
	if strings.Contains(chartDep.Name, "grafana") {
		repo = upstreamGrafanaChartsUrl
	}

	found, tags := git.FindTagsMatching(repo, tag)
	if !found {
		panic("Could not find any tags for this chart")
	}

	highestTag := git.FindHighestVersionTag(tags, chartDep.Name)

	return found, highestTag
}

func findNewestReleaseTagInfo(chartDep rebase.ChartDep) *DependencyChartVersion {
	exists, tag := findNewestReleaseTag(chartDep)
	if !exists {
		return nil
	}
	return &DependencyChartVersion{
		Name:    chartDep.Name,
		Version: tag.Name().String(),
		Hash:    tag.Hash().String(),
	}
}

func (s *RebaseInfo) FindChartsContainers() error {
	return nil
}

func (s *RebaseInfo) SaveStateToRebaseYaml(saveDir string) {
	yamlData, err := yaml.Marshal(s)
	if err != nil {
		log.Fatalf("Error marshaling YAML: %v", err)
	}

	err = os.WriteFile(fmt.Sprintf("%s/rebase.yaml", saveDir), yamlData, 0644)
	if err != nil {
		log.Fatalf("Error writing YAML to file: %v", err)
	}
}
