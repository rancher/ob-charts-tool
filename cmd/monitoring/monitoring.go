package monitoring

import (
	"fmt"

	"github.com/rancher/ob-charts-tool/cmd/groups"
	"github.com/spf13/cobra"
)

func subCommandList() []*cobra.Command {
	return []*cobra.Command{
		getRebaseInfoCmd,
		testNewVersionCmd,
	}
}

func RegisterMonitoringSubcommands(cmd *cobra.Command) {
	for _, subCmd := range subCommandList() {
		subCmd.Use = fmt.Sprintf("%s:%s", groups.MonitoringGroup.ID, subCmd.Use)
		cmd.AddCommand(subCmd)
	}
}
