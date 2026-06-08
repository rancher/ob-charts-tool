package upstream

// Repository represents a known Helm chart repository.
type Repository string

const (
	RepositoryGrafana    Repository = "https://github.com/grafana-community/helm-charts.git"
	RepositoryPrometheus Repository = "https://github.com/prometheus-community/helm-charts.git"
)
