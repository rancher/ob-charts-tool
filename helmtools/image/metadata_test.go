package image_test

import (
	"testing"

	"github.com/rancher/ob-charts-tool/helmtools/image"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractImagesWithSources(t *testing.T) {
	tests := []struct {
		name        string
		valuesYAML  string
		source      string
		defaultTag  string
		wantCount   int
		wantImage   string
		wantSources []string
		wantOS      string
	}{
		{
			name: "single image with source",
			valuesYAML: `
image:
  registry: docker.io
  repository: nginx
  tag: 1.21.0
`,
			source:      "nginx-chart:1.0.0",
			defaultTag:  "",
			wantCount:   1,
			wantImage:   "docker.io/nginx:1.21.0",
			wantSources: []string{"nginx-chart:1.0.0"},
			wantOS:      "linux",
		},
		{
			name: "windows image detection",
			valuesYAML: `
image:
  repository: mcr.microsoft.com/windows/nanoserver
  tag: ltsc2022
`,
			source:      "windows-chart:1.0.0",
			defaultTag:  "",
			wantCount:   1,
			wantImage:   "mcr.microsoft.com/windows/nanoserver:ltsc2022",
			wantSources: []string{"windows-chart:1.0.0"},
			wantOS:      "windows",
		},
		{
			name: "default tag applied",
			valuesYAML: `
image:
  repository: alpine
  tag: ""
`,
			source:      "alpine-chart:1.0.0",
			defaultTag:  "v1.2.3",
			wantCount:   1,
			wantImage:   "alpine:v1.2.3",
			wantSources: []string{"alpine-chart:1.0.0"},
			wantOS:      "linux",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			refs, err := image.ExtractImagesWithSources([]byte(tt.valuesYAML), tt.source, tt.defaultTag)
			require.NoError(t, err)

			assert.Len(t, refs, tt.wantCount)

			if tt.wantImage != "" {
				ref, exists := refs[tt.wantImage]
				require.True(t, exists, "expected image %s not found", tt.wantImage)
				assert.Equal(t, tt.wantSources, ref.Sources)
				assert.Equal(t, tt.wantOS, ref.OS)
				assert.Equal(t, tt.wantImage, ref.FullImage())
			}
		})
	}
}

func TestExtractImagesWithSources_MultipleImages(t *testing.T) {
	valuesYAML := `
app:
  image:
    repository: myapp
    tag: v1.0.0

sidecar:
  image:
    repository: sidecar
    tag: v2.0.0
`

	refs, err := image.ExtractImagesWithSources([]byte(valuesYAML), "multi-chart:1.0.0", "")
	require.NoError(t, err)

	assert.Len(t, refs, 2)
	assert.Contains(t, refs, "myapp:v1.0.0")
	assert.Contains(t, refs, "sidecar:v2.0.0")
}

func TestMergeImageSources(t *testing.T) {
	// First chart
	refs1, err := image.ExtractImagesWithSources([]byte(`
image:
  repository: nginx
  tag: 1.21.0
`), "chart-a:1.0.0", "")
	require.NoError(t, err)

	// Second chart with same image
	refs2, err := image.ExtractImagesWithSources([]byte(`
image:
  repository: nginx
  tag: 1.21.0
`), "chart-b:2.0.0", "")
	require.NoError(t, err)

	// Third chart with different image
	refs3, err := image.ExtractImagesWithSources([]byte(`
image:
  repository: alpine
  tag: 3.14
`), "chart-c:3.0.0", "")
	require.NoError(t, err)

	merged := image.MergeImageSources(refs1, refs2, refs3)

	assert.Len(t, merged, 2)

	nginxRef := merged["nginx:1.21.0"]
	require.NotNil(t, nginxRef)
	assert.ElementsMatch(t, []string{"chart-a:1.0.0", "chart-b:2.0.0"}, nginxRef.Sources)

	alpineRef := merged["alpine:3.14"]
	require.NotNil(t, alpineRef)
	assert.Equal(t, []string{"chart-c:3.0.0"}, alpineRef.Sources)
}

func TestMergeImageSources_NoDuplicateSources(t *testing.T) {
	refs1, err := image.ExtractImagesWithSources([]byte(`
image:
  repository: nginx
  tag: 1.21.0
`), "chart-a:1.0.0", "")
	require.NoError(t, err)

	// Merge same map twice
	merged := image.MergeImageSources(refs1, refs1)

	nginxRef := merged["nginx:1.21.0"]
	require.NotNil(t, nginxRef)
	// Should not have duplicates
	assert.Equal(t, []string{"chart-a:1.0.0"}, nginxRef.Sources)
}

func TestImageReference_FullImage(t *testing.T) {
	tests := []struct {
		name string
		ref  image.ImageReference
		want string
	}{
		{
			name: "with registry",
			ref: image.ImageReference{
				Image: image.Image{
					Registry:   "docker.io",
					Repository: "nginx",
					Tag:        "1.21.0",
				},
			},
			want: "docker.io/nginx:1.21.0",
		},
		{
			name: "without registry",
			ref: image.ImageReference{
				Image: image.Image{
					Repository: "nginx",
					Tag:        "1.21.0",
				},
			},
			want: "nginx:1.21.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.ref.FullImage())
		})
	}
}
