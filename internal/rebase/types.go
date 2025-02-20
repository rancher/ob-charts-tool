package rebase

import "github.com/mallardduck/ob-charts-tool/internal/util"

type ChartDep struct {
	Name       string `yaml:"name"`
	Version    string `yaml:"version"`
	Repository string `yaml:"repository"`
}

type Chart struct {
	Dependencies []ChartDep `yaml:"dependencies"`
}

type FoundChart struct {
	Name         string `yaml:"name"`
	ChartFileURL string `yaml:"chart_file_url"`
	Ref          string `yaml:"ref"`
	CommitHash   string `yaml:"commit_hash"`
	AppVersion   string `yaml:"app_version"`
}

type StartRequest struct {
	TargetVersion     string
	targetChart       []byte
	FoundChart        FoundChart
	ChartDependencies []ChartDep
}

type ChartRebaseInfo struct {
	TargetVersion           string                          `yaml:"target_version"`
	FoundChart              FoundChart                      `yaml:"found_chart"`
	ChartDependencies       []ChartDep                      `yaml:"chart_dependencies"`
	DependencyChartVersions []DependencyChartVersion        `yaml:"dependency_chart_versions"`
	ChartsImagesLists       map[string]util.Set[ChartImage] `yaml:"charts_images_lists"`
}

type DependencyChartVersion struct {
	Name       string `yaml:"name"`
	Ref        string `yaml:"ref"`
	CommitHash string `yaml:"hash"`
	ChartURL   string `yaml:"chart_url"`
}

type ChartImage struct {
	Registry   string `yaml:"registry"`
	Repository string `yaml:"repository"`
	Tag        string `yaml:"tag"`
	SHA        string `yaml:"sha"`
}
