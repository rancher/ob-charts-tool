package image

import (
	"regexp"
	"strings"

	"github.com/rancher/ob-charts-tool/helmtools/util"
)

// ExtractImagesFromTemplates extracts image references from rendered Helm templates.
// This uses regex pattern matching to find image: and docker.io references.
// Returns a set of image strings (e.g., "rancher/image:tag").
func ExtractImagesFromTemplates(renderedChart string) util.Set[string] {
	imagesSet := util.NewSet[string]()

	// Find "image: ..." patterns
	re := regexp.MustCompile(`image: (.*)`)
	imageList := re.FindAllString(renderedChart, -1)
	for _, image := range imageList {
		imagesSet.Add(image)
	}

	// Find "docker.io" patterns, excluding registry: lines
	re = regexp.MustCompile(`(.*)docker.io(.*)`)
	imageList = re.FindAllString(renderedChart, -1)
	imageList = util.FilterSlice(imageList, func(s string) bool {
		return !strings.Contains(strings.ToLower(s), "registry:")
	})
	for _, image := range imageList {
		imagesSet.Add(image)
	}

	// Clean up image strings
	imagesSet = imagesSet.Map(func(s string) string {
		// Remove assignment prefix (e.g., "IMAGE=...")
		if strings.Contains(s, "=") {
			s = strings.Split(s, "=")[1]
		}
		// Extract value after space (excluding template variables)
		if strings.Contains(s, " ") && !strings.Contains(s, "{{") {
			s = strings.Split(s, " ")[1]
		}
		return s
	}).Map(func(s string) string {
		// Remove quotes and docker.io/ prefix
		s = strings.ReplaceAll(s, "\"", "")
		if strings.Index(s, "docker.io/") == 0 {
			s = strings.ReplaceAll(s, "docker.io/", "")
		}
		return s
	})

	return imagesSet
}
