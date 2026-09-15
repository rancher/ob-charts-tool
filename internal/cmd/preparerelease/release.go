package preparerelease

import (
	"fmt"
	"os"
	"path"

	"go.yaml.in/yaml/v3"
)

func FilterORBSCharts(filter func(string) bool, chartDir string) map[string][]string {
	releaseList, err := readReleaseYaml(chartDir)
	if err != nil {
		// TODO better error handling
		panic(err)
	}

	orbsReleaseItems := make(releaseConfig, 1)
	for chart, versions := range *releaseList {
		if filter(chart) {
			orbsReleaseItems[chart] = versions
		}
	}

	return orbsReleaseItems
}

type releaseConfig map[string][]string

func readReleaseYaml(chartDir string) (*releaseConfig, error) {
	releaseYamlPath := path.Join(chartDir, "release.yaml")
	releaseYaml, err := os.OpenFile(releaseYamlPath, os.O_RDONLY, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to open release.yaml file: %w", err)
	}
	defer releaseYaml.Close()

	var release releaseConfig

	decoder := yaml.NewDecoder(releaseYaml)
	if err := decoder.Decode(&release); err != nil {
		return nil, fmt.Errorf("failed to decode yaml: %w", err)
	}

	return &release, nil
}
