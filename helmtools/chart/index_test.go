package chart_test

import (
	"testing"

	"github.com/rancher/ob-charts-tool/helmtools/chart"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseIndex(t *testing.T) {
	indexYAML := `
apiVersion: v1
entries:
  nginx:
    - name: nginx
      version: 1.0.0
      description: NGINX web server
      appVersion: "1.21.0"
      urls:
        - nginx-1.0.0.tgz
    - name: nginx
      version: 0.9.0
      description: NGINX web server
      appVersion: "1.20.0"
      urls:
        - nginx-0.9.0.tgz
  postgresql:
    - name: postgresql
      version: 11.6.12
      description: PostgreSQL database
      appVersion: "14.5"
      urls:
        - postgresql-11.6.12.tgz
generated: 2023-06-08T12:00:00Z
`

	index, err := chart.ParseIndex([]byte(indexYAML))
	require.NoError(t, err)

	assert.Equal(t, "v1", index.APIVersion)
	assert.Len(t, index.Entries, 2)
	assert.Contains(t, index.Entries, "nginx")
	assert.Contains(t, index.Entries, "postgresql")
}

func TestParseIndex_Empty(t *testing.T) {
	_, err := chart.ParseIndex([]byte{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "index data cannot be empty")
}

func TestParseIndex_Invalid(t *testing.T) {
	_, err := chart.ParseIndex([]byte("not yaml"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse index.yaml")
}

func TestIndex_GetLatestVersion(t *testing.T) {
	indexYAML := `
apiVersion: v1
entries:
  nginx:
    - name: nginx
      version: 1.0.0
      appVersion: "1.21.0"
      urls:
        - nginx-1.0.0.tgz
    - name: nginx
      version: 0.9.0
      appVersion: "1.20.0"
      urls:
        - nginx-0.9.0.tgz
  postgresql:
    - name: postgresql
      version: 11.6.12
      appVersion: "14.5"
      urls:
        - postgresql-11.6.12.tgz
`

	index, err := chart.ParseIndex([]byte(indexYAML))
	require.NoError(t, err)

	tests := []struct {
		name           string
		chartName      string
		wantVersion    string
		wantAppVersion string
		wantNil        bool
	}{
		{
			name:           "get latest nginx",
			chartName:      "nginx",
			wantVersion:    "1.0.0",
			wantAppVersion: "1.21.0",
			wantNil:        false,
		},
		{
			name:           "get latest postgresql",
			chartName:      "postgresql",
			wantVersion:    "11.6.12",
			wantAppVersion: "14.5",
			wantNil:        false,
		},
		{
			name:      "chart not found",
			chartName: "nonexistent",
			wantNil:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			latest := index.GetLatestVersion(tt.chartName)
			if tt.wantNil {
				assert.Nil(t, latest)
			} else {
				require.NotNil(t, latest)
				assert.Equal(t, tt.wantVersion, latest.Version)
				assert.Equal(t, tt.wantAppVersion, latest.AppVersion)
			}
		})
	}
}

func TestIndex_GetChartVersions(t *testing.T) {
	indexYAML := `
apiVersion: v1
entries:
  nginx:
    - name: nginx
      version: 1.0.0
    - name: nginx
      version: 0.9.0
    - name: nginx
      version: 0.8.0
  single:
    - name: single
      version: 1.0.0
`

	index, err := chart.ParseIndex([]byte(indexYAML))
	require.NoError(t, err)

	tests := []struct {
		name      string
		chartName string
		wantCount int
	}{
		{
			name:      "multiple versions",
			chartName: "nginx",
			wantCount: 3,
		},
		{
			name:      "single version",
			chartName: "single",
			wantCount: 1,
		},
		{
			name:      "chart not found",
			chartName: "nonexistent",
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			versions := index.GetChartVersions(tt.chartName)
			assert.Len(t, versions, tt.wantCount)
		})
	}
}

func TestIndex_ListCharts(t *testing.T) {
	indexYAML := `
apiVersion: v1
entries:
  nginx:
    - name: nginx
      version: 1.0.0
  postgresql:
    - name: postgresql
      version: 11.6.12
  redis:
    - name: redis
      version: 17.0.0
`

	index, err := chart.ParseIndex([]byte(indexYAML))
	require.NoError(t, err)

	charts := index.ListCharts()
	assert.Len(t, charts, 3)
	assert.ElementsMatch(t, []string{"nginx", "postgresql", "redis"}, charts)
}

func TestIndex_ListCharts_Empty(t *testing.T) {
	indexYAML := `
apiVersion: v1
entries: {}
`

	index, err := chart.ParseIndex([]byte(indexYAML))
	require.NoError(t, err)

	charts := index.ListCharts()
	assert.Len(t, charts, 0)
}

func TestIndexEntry_AllFields(t *testing.T) {
	indexYAML := `
apiVersion: v1
entries:
  nginx:
    - name: nginx
      version: 1.0.0
      description: NGINX web server
      apiVersion: v2
      appVersion: "1.21.0"
      type: application
      urls:
        - https://charts.example.com/nginx-1.0.0.tgz
      created: 2023-06-08T12:00:00Z
      digest: sha256:abc123
      keywords:
        - web
        - server
      maintainers:
        - name: John Doe
          email: john@example.com
          url: https://example.com
      home: https://nginx.org
      sources:
        - https://github.com/nginx/nginx
      icon: https://nginx.org/icon.png
      deprecated: false
      annotations:
        category: WebServer
`

	index, err := chart.ParseIndex([]byte(indexYAML))
	require.NoError(t, err)

	entry := index.GetLatestVersion("nginx")
	require.NotNil(t, entry)

	assert.Equal(t, "nginx", entry.Name)
	assert.Equal(t, "1.0.0", entry.Version)
	assert.Equal(t, "NGINX web server", entry.Description)
	assert.Equal(t, "v2", entry.APIVersion)
	assert.Equal(t, "1.21.0", entry.AppVersion)
	assert.Equal(t, "application", entry.Type)
	assert.Equal(t, []string{"https://charts.example.com/nginx-1.0.0.tgz"}, entry.URLs)
	assert.Equal(t, "sha256:abc123", entry.Digest)
	assert.Equal(t, []string{"web", "server"}, entry.Keywords)
	assert.Len(t, entry.Maintainers, 1)
	assert.Equal(t, "John Doe", entry.Maintainers[0].Name)
	assert.Equal(t, "https://nginx.org", entry.Home)
	assert.Equal(t, []string{"https://github.com/nginx/nginx"}, entry.Sources)
	assert.Equal(t, "https://nginx.org/icon.png", entry.Icon)
	assert.False(t, entry.Deprecated)
	assert.Equal(t, "WebServer", entry.Annotations["category"])
}
