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

type StartRequest struct {
	TargetVersion     string
	TargetCommitHash  string
	ChartFileURL      string
	targetChart       []byte
	ChartDependencies []ChartDep
	AppVersion        string
}

func (s *StartRequest) FetchChart() {
	s.ChartFileURL = fmt.Sprintf(upstreamChartURL, s.TargetCommitHash)
	resp, err := http.Get(s.ChartFileURL)
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

	s.AppVersion = chart.AppVersion
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
