package syncgrafana

import (
	"bytes"
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

var ReplacementMap = []types.DashboardReplacementRule{
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

func DashboardsSync(tempDir string) error {
	logrus.Info("Syncing grafana dashboards")
	chartPath := config.GetContext().ChartRootDir

	chartsSources := constants.DashboardsSourceCharts()
	for _, chart := range chartsSources {
		currentState := chartState{}
		switch c := chart.(type) {
		case types.DashboardGitSource:
			err := prepareGitDashboard(&currentState, tempDir, c, chartPath)
			if err != nil {
				return err
			}
			common.SetDefaultMaxK8s(&c)
			writeErr := writeOutput(currentState, &c)
			if writeErr != nil {
				return writeErr
			}
		case types.DashboardURLSource:
			err := prepareUrlDashboard(&currentState, c)
			if err != nil {
				return err
			}
			common.SetDefaultMaxK8s(&c)
			writeErr := writeOutput(currentState, &c)
			if writeErr != nil {
				return writeErr
			}
		case types.DashboardFileSource:
			// Needs to be essentially: https://github.com/prometheus-community/helm-charts/blob/0b60795bb66a21cd368b657f0665d67de3e49da9/charts/kube-prometheus-stack/hack/sync_grafana_dashboards.py#L320
			err := prepareFileDashboard(&currentState, c, chartPath)
			if err != nil {
				return err
			}
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

func PatchDashboardJson(inputContent string, key string) string {
	content := strings.TrimSpace(inputContent)

	var data map[string]interface{}
	err := json.Unmarshal([]byte(content), &data)
	if err != nil {
		return "{{`" + content + "`}}"
	}

	// multicluster
	templating, templatingOk := data["templating"].(map[string]interface{})
	if !templatingOk {
		return "{{`" + content + "`}}"
	}
	list, listOk := templating["list"].([]interface{})
	if !listOk {
		return "{{`" + content + "`}}"
	}

	overwriteList := make([]interface{}, 0)
	for _, item := range list {
		if variable, ok := item.(map[string]interface{}); ok {
			if name, ok := variable["name"].(string); ok && name == "cluster" {
				variable["allValue"] = ".*"
				variable["hide"] = ":multicluster:"
			}
			overwriteList = append(overwriteList, variable)
		} else {
			return "{{`" + content + "`}}"
		}
	}
	templating["list"] = overwriteList
	data["templating"] = templating

	updated := replaceNestedKey(data, "decimals", -1, nil)

	var b bytes.Buffer
	encErr := customJsonEncoder(&b).Encode(updated)
	if encErr != nil {
		return "{{`" + content + "`}}"
	}
	content = b.String()
	replacementString := fmt.Sprintf("`}}{{ if %s }}0{{ else }}2{{ end }}{{`", key)
	content = strings.Replace(content, `":multicluster:"`, replacementString, -1) // this changes things to escaped utf8

	for _, rule := range ReplacementMap {
		content = strings.Replace(content, rule.Match, rule.Replacement, -1)
	}

	content = strings.TrimSpace(content)

	return "{{`" + content + "`}}"
}

func customJsonEncoder(b *bytes.Buffer) *json.Encoder {
	// Use a custom Encoder to disable HTML escaping
	enc := json.NewEncoder(b)
	enc.SetEscapeHTML(false) // Disable HTML escaping
	return enc
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
