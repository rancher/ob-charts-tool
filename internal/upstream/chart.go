package upstream

import (
	"fmt"
	"os"
	"strings"

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/mallardduck/ob-charts-tool/internal/git"
	"github.com/mallardduck/ob-charts-tool/internal/rebase"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

const (
	upstreamGrafanaChartsURL    = "https://github.com/grafana/helm-charts.git"
	upstreamPrometheusChartsURL = "https://github.com/prometheus-community/helm-charts.git"
	upstreamVersionTemplate     = "kube-prometheus-stack-%s"
)

func ChartVersionExists(version string) (bool, string, string) {
	return git.VerifyTagExists(upstreamPrometheusChartsURL, fmt.Sprintf(upstreamVersionTemplate, version))
}

func PrepareRebaseRequestInfo(version string, tagRef string, gitHash string) rebase.StartRequest {
	rebaseRequest := rebase.StartRequest{
		TargetVersion:    version,
		TargetTagRef:     tagRef,
		TargetCommitHash: gitHash,
	}

	rebaseRequest.FetchChart()
	rebaseRequest.FindAppVersion()
	rebaseRequest.FindChartDeps()

	return rebaseRequest
}

type FoundChart struct {
	ChartFileURL     string `yaml:"chart_file_url"`
	Ref              string `yaml:"ref"`
	TargetCommitHash string `yaml:"commit_hash"`
	AppVersion       string `yaml:"app_version"`
}

type RebaseInfo struct {
	TargetVersion           string                   `yaml:"target_version"`
	FoundChart              FoundChart               `yaml:"found_chart"`
	ChartDependencies       []rebase.ChartDep        `yaml:"chart_dependencies"`
	DependencyChartVersions []DependencyChartVersion `yaml:"dependency_chart_versions"`
	ChartsImagesLists       map[string][]string      `yaml:"charts_images_lists"`
}

type DependencyChartVersion struct {
	Name string `yaml:"name"`
	Ref  string `yaml:"ref"`
	Hash string `yaml:"hash"`
}

func CollectRebaseChartsInfo(request rebase.StartRequest) RebaseInfo {
	rebaseInfo := RebaseInfo{
		TargetVersion: request.TargetVersion,
		FoundChart: FoundChart{
			ChartFileURL:     request.ChartFileURL,
			Ref:              request.TargetTagRef,
			TargetCommitHash: request.TargetCommitHash,
			AppVersion:       request.AppVersion,
		},
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

	repo := upstreamPrometheusChartsURL
	tag := fmt.Sprintf("%s-%s", chartDep.Name, version)
	if strings.Contains(chartDep.Name, "grafana") {
		repo = upstreamGrafanaChartsURL
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
		Name: chartDep.Name,
		Ref:  tag.Name().String(),
		Hash: tag.Hash().String(),
	}
}

func (s *RebaseInfo) FindChartsContainers() error {
	// TODO: look up main charts values file and find images from there
	for _, item := range s.DependencyChartVersions {
		// TODO: find each dependency chart's Chart.yaml and values.yaml file
		fmt.Println(item.Name + "@" + item.Hash)
	}
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
