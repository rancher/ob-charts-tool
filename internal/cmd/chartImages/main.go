package chartImages

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/mallardduck/ob-charts-tool/internal/util"

	"github.com/jedib0t/go-pretty/list"
	log "github.com/sirupsen/logrus"
)

type ImageLists struct {
	NeedsManualCheck util.Set[string]
	RancherImages    util.Set[string]
	NonRancherImages util.Set[string]
}

func newImageLists() ImageLists {
	return ImageLists{
		util.NewSet[string](),
		util.NewSet[string](),
		util.NewSet[string](),
	}
}

func PrepareChartImagesList(chart string) ImageLists {
	imageListRes := newImageLists()

	log.Debug("Looking for images...with `image:` strings")
	// Process the input to extract unique image:tag values
	re := regexp.MustCompile(`image: (.*)`)
	imageList := re.FindAllString(chart, -1)
	imagesSet := util.NewSet[string]()
	for _, image := range imageList {
		_ = imagesSet.Add(image)
	}

	log.Debug("Looking for images...with `docker.io` strings")
	// Process the input to extract unique image:tag values
	re = regexp.MustCompile(`(.*)docker.io(.*)`)
	imageList = re.FindAllString(chart, -1)
	imageList = util.FilterSlice(imageList, func(s string) bool {
		return !strings.Contains(strings.ToLower(s), "registry:")
	})
	for _, image := range imageList {
		_ = imagesSet.Add(image)
	}

	imagesSet = imagesSet.Map(func(s string) string {
		if strings.Contains(s, "=") {
			s = strings.Split(s, "=")[1]
		}
		if strings.Contains(s, " ") && !strings.Contains(s, "{{") {
			s = strings.Split(s, " ")[1]
		}
		return s
	}).Map(func(s string) string {
		s = strings.ReplaceAll(s, "\"", "")
		if strings.Index(s, "docker.io/") == 0 {
			s = strings.ReplaceAll(s, "docker.io/", "")
		}
		return s
	})

	// TODO: maybe consider adding image tag check - if non use latest?
	for item := range imagesSet.ValuesChan() {
		if strings.Contains(item, "{{") {
			_ = imageListRes.NeedsManualCheck.Add(item)
			imagesSet.Remove(item)
		}
	}

	for item := range imagesSet.ValuesChan() {
		if strings.Contains(item, "rancher/") {
			_ = imageListRes.RancherImages.Add(item)
			imagesSet.Remove(item)
		}
	}

	for item := range imagesSet.ValuesChan() {
		_ = imageListRes.NonRancherImages.Add(item)
		imagesSet.Remove(item)
	}

	return imageListRes
}

func ProcessRenderedChartImages(chartImages *ImageLists) error {
	if !chartImages.RancherImages.IsEmpty() {
		fmt.Println("\n🩻 We will check these images:")
		l := list.NewWriter()
		l.SetStyle(list.StyleBulletCircle)
		for image := range chartImages.RancherImages {
			l.AppendItem(image)
		}
		fmt.Println(l.Render())
	}

	if !chartImages.NeedsManualCheck.IsEmpty() {
		fmt.Println("\n👨‍🔧 These need manual checks:")
		l := list.NewWriter()
		l.SetStyle(list.StyleBulletCircle)
		for image := range chartImages.NeedsManualCheck {
			l.AppendItem(image)
		}
		fmt.Println(l.Render())
	}

	if !chartImages.NonRancherImages.IsEmpty() {
		fmt.Println("\n💥 These appear to have the wrong source registry:")
		l := list.NewWriter()
		l.SetStyle(list.StyleBulletCircle)
		for image := range chartImages.NonRancherImages {
			l.AppendItem(image)
		}
		fmt.Println(l.Render())
	}

	return nil
}

func CheckRancherImages(rancherImages util.Set[string]) map[string]bool {
	res := make(map[string]bool)
	// Make list of all images and their results,
	// Prepare list to table,
	// Render table
	for item := range rancherImages {
		exists := checkImageTagExists(item)
		res[item] = exists
	}

	return res
}

func checkImageTagExists(image string) bool {
	imageParts := strings.Split(image, ":")
	repo := imageParts[0]
	tag := imageParts[1]

	imageRequestToken := getDockerHubToken(repo)
	return makeImageTagRequest(imageRequestToken, fmt.Sprintf(manifestURLTemplate, repo, tag))
}

const (
	dockerTokenURL      = "https://auth.docker.io/token"
	dockerService       = "registry.docker.io"
	scopeTemplate       = "repository:%s:pull"
	manifestURLTemplate = "https://registry-1.docker.io/v2/%s/manifests/%s"
)

func getDockerHubToken(repo string) string {
	requestScope := fmt.Sprintf(scopeTemplate, repo)
	// Construct query parameters
	params := url.Values{}
	params.Add("service", dockerService)
	params.Add("scope", requestScope)

	// Construct full URL with encoded query parameters
	fullURL := fmt.Sprintf("%s?%s", dockerTokenURL, params.Encode())
	body, err := util.GetHTTPBody(fullURL)
	if err != nil {
		log.Fatal(err)
	}

	responseData := make(map[string]interface{})
	err = json.Unmarshal(body, &responseData)
	if err != nil {
		log.Fatal(err)
	}

	token, ok := responseData["token"]
	if !ok {
		panic(fmt.Errorf("unable to find `token` in the resopnse; requested for `%s`", repo))
	}

	return token.(string)
}

func makeImageTagRequest(token string, url string) bool {
	req, err := http.NewRequest("HEAD", url, nil)
	if err != nil {
		log.Error("Error creating request:", err)
		return false
	}

	req.Header.Add("Accept", "application/vnd.docker.distribution.manifest.v2+json")
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token))

	// Create an HTTP client and send the request
	client := &http.Client{}
	resp, _ := client.Do(req)

	return resp.StatusCode == http.StatusOK
}
