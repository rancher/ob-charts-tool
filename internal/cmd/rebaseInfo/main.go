package rebaseInfo

import (
	"fmt"
	"os"

	"github.com/mallardduck/ob-charts-tool/internal/rebase"

	"github.com/jedib0t/go-pretty/text"
	log "github.com/sirupsen/logrus"
)

func VerifyTagExists(tag string) (string, string) {
	exists := false
	var tagRef string
	var hash string
	if exists, tagRef, hash = rebase.ChartVersionExists(tag); !exists {
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
	rebaseInfoState := rebase.CollectRebaseChartsInfo(rebaseRequest)
	_ = rebaseInfoState.FindChartsContainers()

	return rebaseInfoState
}
