package syncprom

import (
	"fmt"
	"github.com/rancher/ob-charts-tool/internal/cmd/updatemonitoringmixins/common"
	"github.com/rancher/ob-charts-tool/internal/cmd/updatemonitoringmixins/constants"
	"github.com/rancher/ob-charts-tool/internal/cmd/updatemonitoringmixins/types"
	"github.com/sirupsen/logrus"
	"os"
	"regexp"
	"slices"
	"strings"
)

func writeOutput(currentState chartState, chart types.RulesGitSource) error {

	// TODO if alerts has spec get spec.groups, else .groups
	groups := currentState.alerts.Groups
	//fmt.Println(groups)
	for _, group := range groups {
		FixExpr(&group)
		groupName := group.Name

		rulesGroups := []AlertGroup{group}
		rules, yamlErr := common.YamlStrRepr(rulesGroups, 4, true)
		if yamlErr != nil {
			return yamlErr
		}

		initLine := ""
		for _, replaceRule := range ReplacementMap {
			limitGroup := replaceRule.LimitGroup
			if limitGroup == nil || len(limitGroup) == 0 {
				limitGroup = []string{
					groupName,
				}
			}

			if slices.Contains(limitGroup, groupName) && strings.Contains(rules, replaceRule.Match) {
				rules = strings.ReplaceAll(rules, replaceRule.Match, replaceRule.Replacement)
				if replaceRule.Init != "" {
					initLine += "\n" + replaceRule.Init
				}
			}
		}
		// Now append per-alert rules
		rules = AddCustomLabels(rules, group)             // rules = add_custom_labels(rules, group)
		rules = AddCustomAnnotations(rules, group)        // rules = add_custom_annotations(rules, group)
		rules = AddCustomKeepFiringFor(rules)             // rules = add_custom_keep_firing_for(rules)
		rules = AddCustomFor(rules)                       // rules = add_custom_for(rules)
		rules = AddCustomSeverity(rules)                  // rules = add_custom_severity(rules)
		rules = AddRulesConditionsFromConditionMap(rules) // rules = add_rules_conditions_from_condition_map(rules)
		rules = AddRulesPerRuleConditions(rules, group)   // rules = add_rules_per_rule_conditions(rules, group)
		writeErr := writeGroupToFile(groupName, rules, currentState.url, chart.GetDestination(), initLine, chart.GetMinKubernetes(), chart.GetMaxKubernetes())
		if writeErr != nil {
			return writeErr
		}
	}

	return nil
}

func writeGroupToFile(
	resourceName string,
	content string,
	url string,
	destination string,
	initLine string,
	minKubernetesVersion string,
	maxKubernetesVersion string,
) error {
	condition, _ := constants.RulesConditionMap[resourceName]
	headerData := types.HeaderData{
		Name:           strings.ToLower(strings.ReplaceAll(resourceName, "_", "-")),
		URL:            url,
		Condition:      condition,
		InitLine:       initLine,
		MinKubeVersion: minKubernetesVersion,
		MaxKubeVersion: maxKubernetesVersion,
	}

	preparedContent, headerErr := constants.NewRuleHeader(headerData)
	if headerErr != nil {
		panic(headerErr)
	}

	content = FixGroupsIndent(content)

	// Adjust rules
	re := regexp.MustCompile(`\s(?i)(by|on) ?\(`)
	replacement := ` ${1} ({{ range $.Values.defaultRules.additionalAggregationLabels }}{{ . }},{{ end }}`
	preparedContent += re.ReplaceAllString(content, replacement)

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

	logrus.Infof("Generated %s", newFilename)

	return nil
}
