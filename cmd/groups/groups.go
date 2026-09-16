package groups

import "github.com/spf13/cobra"

var ChartsGroup cobra.Group = cobra.Group{
	ID:    "charts",
	Title: "Rancher Charts Commands:",
}

var MonitoringGroup cobra.Group = cobra.Group{
	ID:    "monitoring",
	Title: "Monitoring Commands:",
}

var LoggingGroup cobra.Group = cobra.Group{
	ID:    "logging",
	Title: "Logging Commands:",
}
