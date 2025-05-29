package static

import (
	"github.com/goccy/go-yaml/parser"
	"github.com/sirupsen/logrus"
	yamlv3 "gopkg.in/yaml.v3"
	"os"
	"path/filepath"
	"strings"

	"github.com/goccy/go-yaml"
	"github.com/goccy/go-yaml/ast"
)

var defaultRules = []AppVersionRule{
	{
		ValuesKey: ".Values.image.tag",
	},
}

type ChartMetadata struct {
	Name       string `yaml:"name"`
	Version    string `yaml:"version"`
	AppVersion string `yaml:"appVersion"`
}

func ProcessCharts(workingPath string) error {
	for chartName, enabled := range AppVersionEnabled {
		if !enabled {
			continue
		}

		localSubChartPath, pathErr := chartName.IdentifyLocalPath(workingPath)
		if pathErr != nil {
			logrus.Warn(pathErr)
			continue
		}

		rules := defaultRules
		if AppVersionRules.SubChartHasRules(chartName) {
			rules = AppVersionRules.GetSubChartRules(chartName)
		}

		// TODO: in addition to static rules, we can support "manual replacements via config"
		// Place a images.yaml in the package and we'll read that to use as values/rules...

		logrus.Infof("Processing chart: %s", chartName)
		chartInfo, err := readChartYaml(localSubChartPath)
		if err != nil {
			return err
		}

		// valuesYaml, err := readValuesYaml(subChartPath)
		valuesYamlPath := filepath.Join(localSubChartPath, "charts", "values.yaml")

		valuesFile, parseErr := parser.ParseFile(valuesYamlPath, parser.ParseComments)
		if parseErr != nil {
			panic(parseErr)
		}

		for _, rule := range rules {
			updateValue := chartInfo.AppVersion
			if rule.PrepareFunc != nil {
				updateValue = rule.PrepareFunc(chartInfo.AppVersion)
			}
			logrus.Infof("res: %s", updateValue)

			yamlPathString := strings.Replace(rule.ValuesKey, ".Values", "$", 1)
			// YAMLPath to modify (e.g. $.replicaCount)
			yamlPath, yamlErr := yaml.PathString(yamlPathString)
			if yamlErr != nil {
				panic(err)
			}

			node, readErr := yamlPath.ReadNode(valuesFile)
			if readErr != nil {
				panic(yamlErr)
			}
			logrus.Info(node.String())
			logrus.Info(node)

			newToken := node.GetToken().Clone()
			newToken.Value = updateValue + "-dan.0"
			newNode := ast.String(newToken)
			logrus.Info(newToken)
			logrus.Info(newNode)

			err = yamlPath.ReplaceWithNode(valuesFile, newNode)
			if err != nil {
				return err
			}
		}
		// TODO: after all rules process, save the values.yaml file
		yamlStr := valuesFile.String()
		logrus.Info(yamlStr)
		valuesBytes := []byte(yamlStr)
		writeErr := os.WriteFile(valuesYamlPath, valuesBytes, os.ModePerm)
		if writeErr != nil {
			panic(writeErr)
		}
	}

	return nil
}

func readChartYaml(subChartPath string) (*ChartMetadata, error) {
	chartYamlPath := filepath.Join(subChartPath, "charts", "Chart.yaml")

	_, err := os.Stat(chartYamlPath)
	if err != nil {
		if os.IsNotExist(err) {
			logrus.Warn("Chart path does not exist")
			return nil, nil
		}
		return nil, err
	}

	chartYamlBytes, err := os.ReadFile(chartYamlPath)
	if err != nil {
		return nil, err
	}
	var chartMetadata ChartMetadata
	err = yamlv3.Unmarshal(chartYamlBytes, &chartMetadata)
	if err != nil {
		return nil, err
	}
	return &chartMetadata, nil
}

func readValuesYaml(subChartPath string) (ast.Node, error) {
	valuesYamlPath := filepath.Join(subChartPath, "charts", "values.yaml")
	// Read original YAML file
	data, err := os.ReadFile(valuesYamlPath)
	if err != nil {
		panic(err)
	}

	// Load into YAML Node tree
	var node ast.Node
	if err := yaml.Unmarshal(data, &node); err != nil {
		panic(err)
	}
	return node, nil
}
