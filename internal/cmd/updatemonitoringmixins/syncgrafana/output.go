package syncgrafana

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/rancher/ob-charts-tool/internal/cmd/updatemonitoringmixins/common"
	"github.com/rancher/ob-charts-tool/internal/cmd/updatemonitoringmixins/constants"
	"github.com/rancher/ob-charts-tool/internal/cmd/updatemonitoringmixins/types"

	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

func writeOutput[T types.DashboardSource](currentState chartState, chart T) error {
	if chart.GetType() == types.DashboardYaml {
		var yamlData map[string]interface{}
		err := yaml.Unmarshal([]byte(currentState.rawText), &yamlData)
		if err != nil {
			return err
		}
		groups, _ := yamlData["items"]
		for _, group := range groups.([]interface{}) {
			fmt.Println(group)
		}
	} else if chart.GetType() == types.DashboardJson {
		resourceName := filepath.Base(currentState.source)
		resourceName = strings.TrimSuffix(resourceName, filepath.Ext(resourceName))

		err := writeGroupToFile(
			resourceName,
			currentState.rawText,
			currentState.url,
			chart.GetDestination(),
			chart.GetMinKubernetes(),
			chart.GetMaxKubernetes(),
			chart.GetMulticlusterKey(),
		)
		if err != nil {
			return err
		}
	}

	// if dasboardSource.GetType() == DashboardYaml {
	// 	// TODO
	// 	logrus.Info("TODO")
	// } else if dasboardSource.GetType() == DashboardJsonnetMixin {
	// 	// TODO
	// 	logrus.Info("TODO")
	// } else if dasboardSource.GetType() == DashboardJson {
	// 	// TODO
	// 	logrus.Info("TODO")
	// }

	return nil
}

func writeGroupToFile(
	resourceName string,
	content string,
	url string,
	destination string,
	minKubernetesVersion string,
	maxKubernetesVersion string,
	multiclusterKey string,
) error {
	condition, ok := constants.ConditionMap[resourceName]
	if !ok {
		panic(errors.New(resourceName + " not found in condition map"))
	}

	headerData := constants.HeaderData{
		Name:           resourceName,
		URL:            url,
		Condition:      condition,
		MinKubeVersion: minKubernetesVersion,
		MaxKubeVersion: maxKubernetesVersion,
	}
	preparedContent, headerErr := constants.NewHeader(headerData)
	if headerErr != nil {
		panic(headerErr)
	}

	// TODO all the patch_ stuff
	content = PatchDashboardJson(content, multiclusterKey)
	content = PatchDashboardJsonSetTimezoneAsVariable(content)
	content = PatchDashboardJsonSetEditableAsVariable(content)
	content = PatchDashboardJsonSetIntervalAsVariable(content)

	fileStruct := map[string]interface{}{
		resourceName + ".json": common.LiteralStr(content),
	}
	yamlString, yamlStrErr := common.YamlStrRepr(fileStruct, 2)
	if yamlStrErr != nil {
		return yamlStrErr
	}
	preparedContent += yamlString
	preparedContent += "{{- end }}"

	filename := resourceName + ".yaml"
	newFilename := fmt.Sprintf("%s/%s", destination, filename)

	// make sure directories to store the file exist
	dirErr := os.MkdirAll(destination, os.ModePerm)
	if dirErr != nil {
		return dirErr
	}

	// Recreate the file
	writeErr := os.WriteFile(newFilename, []byte(preparedContent), 0644)
	if writeErr != nil {
		return writeErr
	}

	logrus.Info("Generated %s", newFilename)

	return nil
}
