package verifysubchartimages

import "github.com/rancher/ob-charts-tool/internal/fs"

type ChartMetadata struct {
	Dir        string `yaml:"dir"`
	Name       string `yaml:"name"`
	Version    string `yaml:"version"`
	AppVersion string `yaml:"appVersion"`
}

type VerifyChartData struct {
	ChartMetadata
	FilesUsingAppVersion []string `yaml:"filesUsingAppVersion"`
	AppVersionReferences []fs.SearchReference
}
