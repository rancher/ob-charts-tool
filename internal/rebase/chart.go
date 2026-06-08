package rebase

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/rancher/ob-charts-tool/helmtools/git"
	"github.com/rancher/ob-charts-tool/helmtools/util"
	"github.com/rancher/ob-charts-tool/helmtools/values"
	"github.com/rancher/ob-charts-tool/internal/config"
	"github.com/rancher/ob-charts-tool/internal/upstream"

	"github.com/go-git/go-git/v5/plumbing"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

func findNewestReleaseTagInfo(chartDep ChartDep) *DependencyChartVersion {
	exists, tag := findNewestReleaseTag(chartDep)
	if !exists {
		return nil
	}

	chartChartURL := upstream.BuildChartYAMLURL(chartDep.Name, "TODO-hash")
	chartVersion, appVersion, err := findChartVersionInfo(chartChartURL)
	if err != nil {
		log.Errorf("Failed to find chart version info for %s: %v", chartDep.Name, err)
		return nil
	}

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

	repo := upstream.IdentifyRepository(chartDep.Name)
	tag := fmt.Sprintf("%s-%s", chartDep.Name, version)

	found, tags, err := git.FindMatchingTags(context.Background(), string(repo), tag)
	if err != nil {
		panic(err)
	}
	if !found {
		panic("Could not find any tags for this chart")
	}

	highestTag := git.FindHighestVersionTag(tags, chartDep.Name)
	if highestTag == nil {
		panic("No valid version tags found")
	}

	return found, &plumbing.Reference{} // TODO: Refactor this function
}

func findChartVersionInfo(chartFileURL string) (string, string, error) {
	body, err := util.GetHTTPBody(context.Background(), nil, chartFileURL)
	if err != nil {
		return "", "", err
	}

	var chartMeta ChartMetaData
	if err := yaml.Unmarshal(body, &chartMeta); err != nil {
		return "", "", err
	}

	return chartMeta.Version, chartMeta.AppVersion, nil
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
	valuesFileURL := upstream.BuildValuesYAMLURL(chartName, commitHash)
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

	err := imageResolver.fetchChartValues(valuesFileURL)
	if err != nil {
		log.Errorf("Failed to fetch chart values from %s: %v", valuesFileURL, err)
		return
	}

	// Use the heuristic sweep for all charts so that every image in values.yaml is
	// captured in rebase.yaml, not just the subset that branchverifycheck happens to
	// verify against appVersion.  Rule-based extraction is the right tool for
	// branchverifycheck (targeted version assertions), but rebase info needs the full
	// picture of all images that may need updating.
	err = imageResolver.extractChartValuesImages()
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

func (cir *chartImagesResolver) fetchChartValues(valuesURL string) error {
	body, err := util.GetHTTPBody(context.Background(), nil, valuesURL)
	if err != nil {
		return err
	}
	cir.chartValuesData = body
	return nil
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

				cir.chartImagesList.Add(img)
			}
		}

		// Recursively process nested structures
		cir.extractChartImages(valueNode)
	}
}

// PopulateSubchartTagExpectations computes the expected image tag values for all
// tracked subcharts and stores them on the struct so they are included in rebase.yaml.
// Call this before SaveStateToRebaseYaml.
func (s *ChartRebaseInfo) PopulateSubchartTagExpectations() {
	s.SubchartTagExpectations = nil
	for _, dep := range s.DependencyChartVersions {
		normalized := values.NormalizeName(dep.Name)
		if !config.SubchartsToCheck[normalized] || dep.AppVersion == "" {
			continue
		}
		expectation := SubchartTagExpectation{
			Name:         dep.Name,
			AppVersion:   dep.AppVersion,
			ExpectedTags: make(map[string]string),
		}
		for _, rule := range values.GetRules(normalized, config.SubchartRules, config.DefaultRules) {
			expectation.ExpectedTags[rule.ValuesKey] = rule.Apply(dep.AppVersion)
		}
		s.SubchartTagExpectations = append(s.SubchartTagExpectations, expectation)
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
