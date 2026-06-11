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

func TestExtractImagesWithSources_ExplicitOSField(t *testing.T) {
	tests := []struct {
		name       string
		valuesYAML string
		wantOS     string
	}{
		{
			name: "explicit windows OS",
			valuesYAML: `
image:
  repository: myapp
  tag: v1.0.0
  os: windows
`,
			wantOS: "windows",
		},
		{
			name: "explicit linux OS",
			valuesYAML: `
image:
  repository: myapp
  tag: v1.0.0
  os: linux
`,
			wantOS: "linux",
		},
		{
			name: "comma-separated OS (picks first)",
			valuesYAML: `
image:
  repository: myapp
  tag: v1.0.0
  os: windows,linux
`,
			wantOS: "windows",
		},
		{
			name: "comma-separated OS reverse (picks first)",
			valuesYAML: `
image:
  repository: myapp
  tag: v1.0.0
  os: linux,windows
`,
			wantOS: "linux",
		},
		{
			name: "no OS field - name-based detection (windows)",
			valuesYAML: `
image:
  repository: mcr.microsoft.com/windows/nanoserver
  tag: ltsc2022
`,
			wantOS: "windows",
		},
		{
			name: "no OS field - name-based detection (linux)",
			valuesYAML: `
image:
  repository: nginx
  tag: 1.21.0
`,
			wantOS: "linux",
		},
		{
			name: "explicit OS overrides name-based (windows name but linux OS)",
			valuesYAML: `
image:
  repository: myapp-windows
  tag: v1.0.0
  os: linux
`,
			wantOS: "linux",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			refs, err := image.ExtractImagesWithSources([]byte(tt.valuesYAML), "test-chart:1.0.0", "")
			require.NoError(t, err)
			require.Len(t, refs, 1)

			// Get the single reference
			var ref *image.ImageReference
			for _, r := range refs {
				ref = r
				break
			}

			assert.Equal(t, tt.wantOS, ref.OS)
		})
	}
}

func TestExtractImagesWithSources_MultipleImagesWithMixedOS(t *testing.T) {
	valuesYAML := `
app:
  image:
    repository: myapp
    tag: v1.0.0
    os: linux

windowsSidecar:
  image:
    repository: sidecar
    tag: v2.0.0
    os: windows
`

	refs, err := image.ExtractImagesWithSources([]byte(valuesYAML), "mixed-chart:1.0.0", "")
	require.NoError(t, err)

	assert.Len(t, refs, 2)
	assert.Equal(t, "linux", refs["myapp:v1.0.0"].OS)
	assert.Equal(t, "windows", refs["sidecar:v2.0.0"].OS)
}

// TestExtractImagesWithSources_RancherCompatibility tests scenarios from rancher/rancher
// to ensure helmtools can replace the manual walkMap/pickImagesFromValuesMap logic.
func TestExtractImagesWithSources_RancherCompatibility(t *testing.T) {
	tests := []struct {
		name        string
		valuesYAML  string
		wantImages  map[string]string // image -> OS
		description string
	}{
		{
			name: "rancher pattern: image with os field",
			valuesYAML: `
image:
  repository: rancher/foo
  tag: v1.0.0
  os: windows
`,
			wantImages: map[string]string{
				"rancher/foo:v1.0.0": "windows",
			},
			description: "Matches r/r: explicit os field",
		},
		{
			name: "rancher pattern: image without os field defaults to linux",
			valuesYAML: `
image:
  repository: rancher/foo
  tag: v1.0.0
`,
			wantImages: map[string]string{
				"rancher/foo:v1.0.0": "linux",
			},
			description: "Matches r/r: no os field means linux by default",
		},
		{
			name: "rancher pattern: multi-os image (windows,linux)",
			valuesYAML: `
image:
  repository: rancher/wins
  tag: v0.1.0
  os: windows,linux
`,
			wantImages: map[string]string{
				"rancher/wins:v0.1.0": "windows", // picks first
			},
			description: "Matches r/r: comma-separated os values",
		},
		{
			name: "rancher pattern: multiple images with different os",
			valuesYAML: `
linuxApp:
  image:
    repository: rancher/linux-app
    tag: v1.0.0
    os: linux

windowsApp:
  image:
    repository: rancher/windows-app
    tag: v1.0.0
    os: windows

defaultApp:
  image:
    repository: rancher/default-app
    tag: v1.0.0
`,
			wantImages: map[string]string{
				"rancher/linux-app:v1.0.0":   "linux",
				"rancher/windows-app:v1.0.0": "windows",
				"rancher/default-app:v1.0.0": "linux", // no os field = linux
			},
			description: "Matches r/r: mixed OS images in same values",
		},
		{
			name: "rancher pattern: nested image structure",
			valuesYAML: `
global:
  cattle:
    systemDefaultRegistry: registry.example.com

components:
  server:
    image:
      repository: rancher/rancher
      tag: v2.7.0
      os: linux,windows
  agent:
    image:
      repository: rancher/rancher-agent
      tag: v2.7.0
      os: linux
`,
			wantImages: map[string]string{
				"rancher/rancher:v2.7.0":       "linux", // picks first from "linux,windows"
				"rancher/rancher-agent:v2.7.0": "linux",
			},
			description: "Matches r/r: nested structure with multiple images",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			refs, err := image.ExtractImagesWithSources([]byte(tt.valuesYAML), "test-chart:1.0.0", "")
			require.NoError(t, err, tt.description)

			// Verify we got the expected images
			assert.Len(t, refs, len(tt.wantImages), tt.description)

			for wantImage, wantOS := range tt.wantImages {
				ref, exists := refs[wantImage]
				require.True(t, exists, "expected image %s not found. %s", wantImage, tt.description)
				assert.Equal(t, wantOS, ref.OS, "OS mismatch for %s. %s", wantImage, tt.description)
			}
		})
	}
}

// TestExtractImagesWithSources_RancherFilteringPattern demonstrates how to filter
// images by OS after extraction (matching r/r's osType filtering pattern).
// Uses SupportsOS() to match rancher's comma-separated OS list behavior.
func TestExtractImagesWithSources_RancherFilteringPattern(t *testing.T) {
	valuesYAML := `
linuxImage:
  image:
    repository: rancher/linux-app
    tag: v1.0.0
    os: linux

windowsImage:
  image:
    repository: rancher/windows-app
    tag: v1.0.0
    os: windows

multiOSImage:
  image:
    repository: rancher/multi-os-app
    tag: v1.0.0
    os: linux,windows

defaultImage:
  image:
    repository: rancher/default-app
    tag: v1.0.0
`

	refs, err := image.ExtractImagesWithSources([]byte(valuesYAML), "test-chart:1.0.0", "")
	require.NoError(t, err)

	// Filter for Linux images (simulates r/r's osType == Linux filtering)
	// Use SupportsOS() to match rancher's behavior with multi-OS images
	linuxImages := make([]string, 0)
	for fullImage, ref := range refs {
		if ref.SupportsOS("linux") {
			linuxImages = append(linuxImages, fullImage)
		}
	}

	// Should get linux-app, multi-os-app, and default-app
	assert.Len(t, linuxImages, 3)
	assert.Contains(t, linuxImages, "rancher/linux-app:v1.0.0")
	assert.Contains(t, linuxImages, "rancher/multi-os-app:v1.0.0") // supports both!
	assert.Contains(t, linuxImages, "rancher/default-app:v1.0.0")

	// Filter for Windows images (simulates r/r's osType == Windows filtering)
	windowsImages := make([]string, 0)
	for fullImage, ref := range refs {
		if ref.SupportsOS("windows") {
			windowsImages = append(windowsImages, fullImage)
		}
	}

	// Should get windows-app AND multi-os-app
	assert.Len(t, windowsImages, 2)
	assert.Contains(t, windowsImages, "rancher/windows-app:v1.0.0")
	assert.Contains(t, windowsImages, "rancher/multi-os-app:v1.0.0") // supports both!
}

// TestExtractImagesWithSources_RancherTestCase2 replicates rancher/rancher's test case #2:
// "Want Windows images" with os: "windows,linux" and osType: Windows should add the image.
func TestExtractImagesWithSources_RancherTestCase2(t *testing.T) {
	// Exact scenario from rancher/rancher/pkg/image/charts_test.go test case #2
	valuesYAML := `
image:
  repository: test-repository
  tag: "1.2.3"
  os: windows,linux
`

	refs, err := image.ExtractImagesWithSources([]byte(valuesYAML), "chart:0.1.2", "")
	require.NoError(t, err)

	// Verify we extracted the image
	require.Len(t, refs, 1)
	ref := refs["test-repository:1.2.3"]
	require.NotNil(t, ref)

	// Verify OSList contains both values
	assert.ElementsMatch(t, []string{"windows", "linux"}, ref.OSList)

	// Verify primary OS is windows (first in list)
	assert.Equal(t, "windows", ref.OS)

	// Verify SupportsOS works for BOTH (this is the key rancher behavior)
	assert.True(t, ref.SupportsOS("windows"), "should support windows")
	assert.True(t, ref.SupportsOS("linux"), "should support linux")
}

// TestExtractImagesWithSources_RancherAllTestCases replicates all rancher test scenarios
func TestExtractImagesWithSources_RancherAllTestCases(t *testing.T) {
	tests := []struct {
		name          string
		valuesYAML    string
		filterForOS   string // simulates osType parameter
		shouldInclude bool   // should image be included for this osType?
		description   string
	}{
		{
			name: "Test #1: Want linux images",
			valuesYAML: `
image:
  repository: test-repository
  tag: "1.2.3"
  os: Linux
`,
			filterForOS:   "linux",
			shouldInclude: true,
			description:   "os: Linux with osType: Linux",
		},
		{
			name: "Test #2: Want Windows images (multi-OS)",
			valuesYAML: `
image:
  repository: test-repository
  tag: "1.2.3"
  os: windows,linux
`,
			filterForOS:   "windows",
			shouldInclude: true,
			description:   "os: windows,linux with osType: Windows",
		},
		{
			name: "Test #3: No images of the given OS",
			valuesYAML: `
image:
  repository: test-repository
  tag: "1.2.3"
  os: linux
`,
			filterForOS:   "windows",
			shouldInclude: false,
			description:   "os: linux with osType: Windows",
		},
		{
			name: "Test #4: No OS provided, default to Linux",
			valuesYAML: `
image:
  repository: test-repository
  tag: "1.2.3"
`,
			filterForOS:   "linux",
			shouldInclude: true,
			description:   "no os field with osType: Linux",
		},
		{
			name: "Test #5: Unsupported OS provided",
			valuesYAML: `
image:
  repository: test-repository
  tag: "1.2.3"
  os: unsupported-os
`,
			filterForOS:   "linux",
			shouldInclude: false,
			description:   "os: unsupported-os with osType: Linux (invalid OS ignored)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			refs, err := image.ExtractImagesWithSources([]byte(tt.valuesYAML), "chart:0.1.2", "")
			require.NoError(t, err, tt.description)

			// Filter images for the requested OS (simulating rancher's osType parameter)
			var matchingImages []string
			for fullImage, ref := range refs {
				if ref.SupportsOS(tt.filterForOS) {
					matchingImages = append(matchingImages, fullImage)
				}
			}

			if tt.shouldInclude {
				assert.Len(t, matchingImages, 1, "%s: expected image to be included", tt.description)
				assert.Contains(t, matchingImages, "test-repository:1.2.3", tt.description)
			} else {
				assert.Len(t, matchingImages, 0, "%s: expected image NOT to be included", tt.description)
			}
		})
	}
}
