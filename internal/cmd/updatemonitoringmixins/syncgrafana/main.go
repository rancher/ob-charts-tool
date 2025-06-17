package syncgrafana

import (
	"encoding/json"
	"fmt"
	"github.com/rancher/ob-charts-tool/internal/cmd/updatemonitoringmixins/common"
	"github.com/rancher/ob-charts-tool/internal/cmd/updatemonitoringmixins/config"
	"github.com/rancher/ob-charts-tool/internal/cmd/updatemonitoringmixins/constants"
	"github.com/rancher/ob-charts-tool/internal/cmd/updatemonitoringmixins/git"
	"github.com/rancher/ob-charts-tool/internal/cmd/updatemonitoringmixins/types"
	"github.com/sirupsen/logrus"
	"reflect"
	"regexp"
	"strings"
)

type ReplacementRule struct {
	Match       string
	Replacement string
}

var ReplacementMap = []ReplacementRule{
	{
		"var-namespace=$__cell_1",
		"var-namespace=`}}{{ if .Values.grafana.sidecar.dashboards.enableNewTablePanelSyntax }}${__data.fields.namespace}{{ else }}$__cell_1{{ end }}{{`",
	},
	{
		"var-type=$__cell_2",
		"var-type=`}}{{ if .Values.grafana.sidecar.dashboards.enableNewTablePanelSyntax }}${__data.fields.workload_type}{{ else }}$__cell_2{{ end }}{{`",
	},
	{
		"=$__cell",
		"=`}}{{ if .Values.grafana.sidecar.dashboards.enableNewTablePanelSyntax }}${__value.text}{{ else }}$__cell{{ end }}{{`",
	},
	{
		`job=\"prometheus-k8s\",namespace=\"monitoring\"`,
		"",
	},
}

func DashboardsSync(tempDir string, repos []git.RepoConfigStatus) error {
	chartPath := config.GetContext().ChartRootDir
	fmt.Println(chartPath)

	// repoSHAs is equivalent to `refs` from `sync_grafana_dashboards.py` or `sync_prometheus_rules.py`
	repoSHAs := git.RepoSHAs(repos)
	chartsSources := constants.SourceCharts(repoSHAs)
	for _, chart := range chartsSources {
		currentState := chartState{}
		switch c := chart.(type) {
		case types.DashboardGitSource:
			fmt.Print("git source", c)
			err := prepareGitDashboard(&currentState, tempDir, c, chartPath)
			if err != nil {
				return err
			}
			common.SetDefaultMaxK8s(&c)
		case types.DashboardURLSource:
			fmt.Print("url source", c)
			err := prepareUrlDashboard(&currentState, c)
			if err != nil {
				return err
			}
			currentState.source = c.Source
			currentState.url = c.Source
			common.SetDefaultMaxK8s(&c)
		case types.DashboardFileSource:
			// Needs to be essentially: https://github.com/prometheus-community/helm-charts/blob/0b60795bb66a21cd368b657f0665d67de3e49da9/charts/kube-prometheus-stack/hack/sync_grafana_dashboards.py#L320
			fmt.Print("file source", c)
			err := prepareFileDashboard(&currentState, c, chartPath)
			if err != nil {
				return err
			}
			currentState.source = c.Source
			currentState.url = c.Source
			common.SetDefaultMaxK8s(&c)
			writeErr := writeOutput(currentState, &c)
			if writeErr != nil {
				return writeErr
			}
		default:
			return fmt.Errorf("unknown chart type: %T", c)
		}
	}

	logrus.Info("Finished syncing grafana dashboards")

	return nil
}

func PatchDashboardJson(content string, key string) string {
	var data interface{}
	err := json.Unmarshal([]byte(content), &data)
	if err != nil {
		panic(err)
	}

	// multicluster
	if mapData, ok := data.(map[string]interface{}); ok {
		if templating, ok := mapData["templating"].(map[string]interface{}); ok {
			if list, ok := templating["list"].([]interface{}); ok {
				overwriteList := make([]interface{}, 0)
				for _, item := range list {
					if variable, ok := item.(map[string]interface{}); ok {
						if name, ok := variable["name"].(string); ok && name == "cluster" {
							variable["allValue"] = ".*"
							variable["hide"] = ":multicluster:"
						}
						overwriteList = append(overwriteList, variable)
					} else {
						overwriteList = append(overwriteList, item) // Keep non-map items as is
					}
				}
				templating["list"] = overwriteList
			}
			mapData["templating"] = templating
		}
	}

	data = replaceNestedKey(data, "decimals", -1, nil)

	contentBytes, err := json.Marshal(data)
	content = string(contentBytes)
	replacementString := fmt.Sprintf("`}}{{ if %s }}0{{ else }}2{{ end }}{{`", key)
	content = strings.Replace(content, "\":multicluster:\"", replacementString, -1)

	for _, rule := range ReplacementMap {
		content = strings.Replace(content, rule.Match, rule.Replacement, -1)
	}

	content = regexp.MustCompile(`\\u0026`).ReplaceAllString(content, "&")

	return "{{`" + content + "`}}"
}

func replaceNestedKey(data interface{}, key string, value interface{}, replace interface{}) interface{} {
	switch v := data.(type) {
	case map[string]interface{}:
		newMap := make(map[string]interface{})
		for k, val := range v {
			if k == key && reflect.DeepEqual(val, value) {
				newMap[k] = replace
			} else {
				newMap[k] = replaceNestedKey(val, key, value, replace)
			}
		}
		return newMap
	case []interface{}:
		newList := make([]interface{}, len(v))
		for i, item := range v {
			newList[i] = replaceNestedKey(item, key, value, replace)
		}
		return newList
	default:
		return data
	}
}

const (
	timezoneReplacement = `"timezone": "` + "`" + `}}{{ .Values.grafana.defaultDashboardsTimezone }}{{` + "`" + `"`
	editableReplacement = `"editable":` + "`" + `}}{{ .Values.grafana.defaultDashboardsEditable }}{{` + "`"
	intervalReplacement = `"interval":"` + "`" + `}}{{ .Values.grafana.defaultDashboardsInterval }}{{` + "`" + `"`
)

func PatchDashboardJsonSetTimezoneAsVariable(content string) string {
	timezoneRegexp := regexp.MustCompile(`"timezone"\s*:\s*"(?:\\.|[^\"])*"`)
	content = timezoneRegexp.ReplaceAllString(content, timezoneReplacement)
	return content
}

func PatchDashboardJsonSetEditableAsVariable(content string) string {
	editableRegexp := regexp.MustCompile(`"editable"\s*:\s*(?:true|false)`)
	content = editableRegexp.ReplaceAllString(content, editableReplacement)
	return content
}

func PatchDashboardJsonSetIntervalAsVariable(content string) string {
	intervalRegexp := regexp.MustCompile(`"interval"\s*:\s*"(?:\\.|[^\"])*"`)
	content = intervalRegexp.ReplaceAllString(content, intervalReplacement)
	return content
}
