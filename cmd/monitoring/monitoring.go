package monitoring

import (
	"fmt"
	"github.com/rancher/ob-charts-tool/cmd/groups"
	"github.com/spf13/cobra"
)

func subCommandList() []*cobra.Command {
	return []*cobra.Command{
		getRebaseInfoCmd,
	}
}

func init() {
	for _, cmd := range subCommandList() {
		cmd.Use = fmt.Sprintf("%s:%s", groups.MonitoringGroup.ID, cmd.Use)
	}
}

func RegisterMonitoringSubcommands(cmd *cobra.Command) {
	for _, subCmd := range subCommandList() {
		cmd.AddCommand(subCmd)
	}
}
