package rebase

import (
	"fmt"
	"github.com/mallardduck/ob-charts-tool/internal/util"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

const (
	upstreamChartURL = "https://raw.githubusercontent.com/prometheus-community/helm-charts/%s/charts/kube-prometheus-stack/Chart.yaml"
)

func PrepareRebaseRequestInfo(version string, tagRef string, gitHash string) StartRequest {
	rebaseRequest := StartRequest{
		TargetVersion: version,
		FoundChart: FoundChart{
			Name:       "kube-prometheus-stack",
			Ref:        tagRef,
			CommitHash: gitHash,
		},
	}

	rebaseRequest.FetchChart()
	rebaseRequest.FindAppVersion()
	rebaseRequest.FindChartDeps()

	return rebaseRequest
}

func (s *StartRequest) FetchChart() {
	s.FoundChart.ChartFileURL = fmt.Sprintf(upstreamChartURL, s.FoundChart.CommitHash)
	body, err := util.GetHTTPBody(s.FoundChart.ChartFileURL)
	if err != nil {
		panic(err)
	}
	s.targetChart = body
}

func (s *StartRequest) FindAppVersion() {
	var chart struct {
		AppVersion string `yaml:"appVersion"`
	}
	err := yaml.Unmarshal(s.targetChart, &chart)
	if err != nil {
		return
	}

	s.FoundChart.AppVersion = chart.AppVersion
}

func (s *StartRequest) FindChartDeps() {
	var chart Chart
	err := yaml.Unmarshal(s.targetChart, &chart)
	if err != nil {
		return
	}

	s.ChartDependencies = util.FilterSlice[ChartDep](chart.Dependencies, func(item ChartDep) bool {
		return item.Name != "crds"
	})
	s.targetChart = nil
}

func (s *StartRequest) CollectRebaseChartsInfo() ChartRebaseInfo {
	rebaseInfo := ChartRebaseInfo{
		TargetVersion:     s.TargetVersion,
		FoundChart:        s.FoundChart,
		ChartDependencies: s.ChartDependencies,
		ChartsImagesLists: make(map[string]util.Set[ChartImage]),
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
