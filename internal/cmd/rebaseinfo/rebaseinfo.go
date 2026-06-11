package rebaseinfo

import (
	"context"
	"fmt"
	"os"

	"github.com/rancher/ob-charts-tool/helmtools/git"
	"github.com/rancher/ob-charts-tool/internal/rebase"
	"github.com/rancher/ob-charts-tool/internal/upstream"

	"github.com/jedib0t/go-pretty/text"
	log "github.com/sirupsen/logrus"
)

func VerifyTagExists(tag string) (string, string) {
	// Construct the full tag name for kube-prometheus-stack
	fullTag := "kube-prometheus-stack-" + tag
	exists, tagRef, hash, err := git.VerifyTagExists(context.Background(), string(upstream.RepositoryPrometheus), fullTag)
	if err != nil || !exists {
		errorText := fmt.Sprintf("Cannot find upstream chart version `%s`", tag)
		fmt.Println(
			text.AlignCenter.Apply(
				text.Color.Sprint(text.FgRed, errorText),
				75,
			),
		)
		log.Error(errorText)
		os.Exit(1)
	}

	return tagRef, hash
}

func CollectInfo(version string, ref string, hash string) rebase.ChartRebaseInfo {
	rebaseRequest := rebase.PrepareRebaseRequestInfo(version, ref, hash)
	rebaseInfoState := rebaseRequest.CollectRebaseChartsInfo()
	_ = rebaseInfoState.FindChartsContainers()
	// TODO: Add something that will actually "resolve the images"
	// This way it can output a clear list of docker images and their tags
	// This means filling in the blanks where tags are empty (likely with appVersion)
	// And also resolving what chart tag is the "latest" version for each chart using that as a rolling tag
	// This way at any time our team does a rebase "latest" resolve to a specific tag for QA to test with.

	return rebaseInfoState
}
