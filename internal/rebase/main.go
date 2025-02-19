package rebase

import (
	"fmt"
	"io"
	"net/http"

	"github.com/mallardduck/ob-charts-tool/internal/util"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

const (
	upstreamChartURL = "https://raw.githubusercontent.com/prometheus-community/helm-charts/%s/charts/kube-prometheus-stack/Chart.yaml"
)

type FoundChart struct {
	Name         string `yaml:"name"`
	ChartFileURL string `yaml:"chart_file_url"`
	Ref          string `yaml:"ref"`
	CommitHash   string `yaml:"commit_hash"`
	AppVersion   string `yaml:"app_version"`
}

type StartRequest struct {
	FoundChart        FoundChart
	TargetVersion     string
	targetChart       []byte
	ChartDependencies []ChartDep
}

func PrepareRebaseRequestInfo(version string, tagRef string, gitHash string) StartRequest {
	rebaseRequest := StartRequest{
		FoundChart: FoundChart{
			Name:       "kube-prometheus-stack",
			Ref:        tagRef,
			CommitHash: gitHash,
		},
		TargetVersion: version,
	}

	rebaseRequest.FetchChart()
	rebaseRequest.FindAppVersion()
	rebaseRequest.FindChartDeps()

	return rebaseRequest
}

func (s *StartRequest) FetchChart() {
	s.FoundChart.ChartFileURL = fmt.Sprintf(upstreamChartURL, s.FoundChart.CommitHash)
	resp, err := http.Get(s.FoundChart.ChartFileURL)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}
	s.targetChart = body
}

type ChartDep struct {
	Name       string `yaml:"name"`
	Version    string `yaml:"version"`
	Repository string `yaml:"repository"`
}

type Chart struct {
	Dependencies []ChartDep `yaml:"dependencies"`
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
