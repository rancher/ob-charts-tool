package rebase

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/rancher/ob-charts-tool/internal/git"
	gitremote "github.com/rancher/ob-charts-tool/internal/git/remote"
	monsubcharts "github.com/rancher/ob-charts-tool/internal/monitoring"
	"github.com/rancher/ob-charts-tool/internal/upstream"
	"github.com/rancher/ob-charts-tool/internal/util"

	"github.com/go-git/go-git/v5/plumbing"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

func findNewestReleaseTagInfo(chartDep ChartDep) *DependencyChartVersion {
	exists, tag := findNewestReleaseTag(chartDep)
	if !exists {
		return nil
	}

	chartChartURL := upstream.GetChartsChartURL(chartDep.Name, tag.Hash().String())
	chartVersion, appVersion := findChartVersionInfo(chartChartURL)

	return &DependencyChartVersion{
		Name:         chartDep.Name,
		Ref:          tag.Name().String(),
		CommitHash:   tag.Hash().String(),
		ChartURL:     chartChartURL,
		ChartVersion: chartVersion,
		AppVersion:   appVersion,
	}
}

func findNewestReleaseTag(chartDep ChartDep) (bool, *plumbing.Reference) {
	version := chartDep.Version
	if strings.Contains(version, ".*") {
		version = strings.ReplaceAll(version, ".*", "")
	}

	repo := upstream.IdentifyChartUpstream(chartDep.Name)
	tag := fmt.Sprintf("%s-%s", chartDep.Name, version)

	found, tags := gitremote.FindTagsMatching(repo, tag)
	if !found {
		panic("Could not find any tags for this chart")
	}

	highestTag := git.FindHighestVersionTag(tags, chartDep.Name)

	return found, highestTag
}

func findChartVersionInfo(chartFileURL string) (string, string) {
	body, err := util.GetHTTPBody(chartFileURL)
	if err != nil {
		panic(err)
	}

	var chartMeta ChartMetaData
	if err := yaml.Unmarshal(body, &chartMeta); err != nil {
		panic(err)
	}

	return chartMeta.Version, chartMeta.AppVersion
}

func (s *ChartRebaseInfo) FindChartsContainers() error {
	log.Info("Finding containers for: " + s.FoundChart.Name + "@" + s.FoundChart.CommitHash)
	s.lookupChartImages(s.FoundChart.Name, s.FoundChart.CommitHash)

	for _, item := range s.DependencyChartVersions {
		log.Info("Finding containers for: " + item.Name + "@" + item.CommitHash)
		s.lookupChartImages(item.Name, item.CommitHash)
	}
	return nil
}

func (s *ChartRebaseInfo) lookupChartImages(chartName string, commitHash string) {
	// TODO: Add output for debug and normal flows
	valuesFileURL := upstream.GetChartValuesURL(chartName, commitHash)
	log.Debugf("Fetching '%s' values file from: %s", chartName, valuesFileURL)

	chartImageSet := make(util.Set[ChartImage])

	imageResolver := chartImagesResolver{
		currentChartName: chartName,
		currentHash:      commitHash,
		chartValuesURL:   valuesFileURL,
		chartImagesList:  &chartImageSet,
	}

	if chartName == "kube-prometheus-stack" {
		imageResolver.chartVersion = s.FoundChart.ChartVersion
		imageResolver.appVersion = s.FoundChart.AppVersion
	} else {
		var chartDep DependencyChartVersion
		for _, item := range s.DependencyChartVersions {
			if item.Name == chartName {
				chartDep = item
				break
			}
		}
		imageResolver.chartVersion = chartDep.ChartVersion
		imageResolver.appVersion = chartDep.AppVersion
	}

	imageResolver.fetchChartValues(valuesFileURL)

	// For subcharts with explicit rules, extract images via the rule-defined paths so the
	// rebase.yaml is driven by the same logic that branchverifycheck uses.
	// For all other charts (e.g. kube-prometheus-stack), fall back to the heuristic sweep.
	var err error
	normalizedName := monsubcharts.NormalizeName(chartName)
	if monsubcharts.SubchartsToCheck[normalizedName] {
		err = imageResolver.extractRuleBasedImages(normalizedName)
	} else {
		err = imageResolver.extractChartValuesImages()
	}
	if err != nil {
		log.Error(err)
		log.Exit(1)
	}
	log.Debugf("'%s' chart has found these images: %v", chartName, chartImageSet.Values())
	s.ChartsImagesLists[chartName] = chartImageSet
}

type chartImagesResolver struct {
	currentChartName string
	currentHash      string
	chartVersion     string
	appVersion       string
	chartValuesURL   string
	chartValuesData  []byte
	chartImagesList  *util.Set[ChartImage]
}

func (cir *chartImagesResolver) fetchChartValues(valuesURL string) {
	body, err := util.GetHTTPBody(valuesURL)
	if err != nil {
		panic(err)
	}
	cir.chartValuesData = body
}

// extractRuleBasedImages extracts image information from values.yaml using the explicit paths
// defined in the subchart rules, keeping the rebase.yaml consistent with what branchverifycheck verifies.
func (cir *chartImagesResolver) extractRuleBasedImages(normalizedChartName string) error {
	var values map[string]interface{}
	if err := yaml.Unmarshal(cir.chartValuesData, &values); err != nil {
		return fmt.Errorf("error parsing values yaml for %s: %w", normalizedChartName, err)
	}

	for _, rule := range monsubcharts.GetRules(normalizedChartName) {
		imageMap, ok := monsubcharts.NavigateYAMLMap(values, rule.ImageMapPath())
		if !ok {
			log.Debugf("'%s': image map not found at path '%s'", normalizedChartName, rule.ImageMapPath())
			continue
		}

		img := ChartImage{
			Registry:   stringFromMap(imageMap, "registry"),
			Repository: stringFromMap(imageMap, "repository"),
			Tag:        stringFromMap(imageMap, "tag"),
		}

		if img.Tag == "" {
			log.Warnf("'%s' image tag at '%s' is empty; using appVersion (%s)", normalizedChartName, rule.ValuesKey, cir.appVersion)
			img.Tag = rule.Apply(cir.appVersion)
		}

		_ = cir.chartImagesList.Add(img)
	}
	return nil
}

func stringFromMap(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func (cir *chartImagesResolver) extractChartValuesImages() error {
	var root yaml.Node
	err := yaml.Unmarshal(cir.chartValuesData, &root)
	if err != nil {
		return fmt.Errorf("error parsing values yaml: %v", err)
	}

	cir.extractChartImages(&root)

	return nil
}

func (cir *chartImagesResolver) extractChartImages(node *yaml.Node) {
	if node == nil {
		return
	}

	// Handle DocumentNode by processing its content
	if node.Kind == yaml.DocumentNode && len(node.Content) > 0 {
		cir.extractChartImages(node.Content[0])
		return
	}

	// Process MappingNode (key-value pairs)
	if node.Kind != yaml.MappingNode {
		return
	}

	imageKeyPattern := regexp.MustCompile(`(?i)^(.+)?image$`)

	for i := 0; i < len(node.Content); i += 2 {
		keyNode := node.Content[i]
		valueNode := node.Content[i+1]

		if keyNode.Kind == yaml.ScalarNode && imageKeyPattern.MatchString(keyNode.Value) {
			var img ChartImage
			if err := valueNode.Decode(&img); err == nil {
				// Verify image version tag is set
				if img.Tag == "" {
					// TODO: Verify this logic works for all tags with empty values
					log.Warnf("The image tag for '%s' (part of %s) is empty and will be set to appVersion (%s) value", img.Repository, cir.currentChartName, cir.appVersion)
					img.Tag = cir.appVersion
				}

				_ = cir.chartImagesList.Add(img)
			}
		}

		// Recursively process nested structures
		cir.extractChartImages(valueNode)
	}
}

func (s *ChartRebaseInfo) SaveStateToRebaseYaml(saveDir string) string {
	yamlData, err := yaml.Marshal(s)
	if err != nil {
		log.Fatalf("Error marshaling YAML: %v", err)
	}

	err = os.WriteFile(fmt.Sprintf("%s/rebase.yaml", saveDir), yamlData, 0644)
	if err != nil {
		log.Fatalf("Error writing YAML to file: %v", err)
	}

	return fmt.Sprintf("%s/rebase.yaml", saveDir)
}
