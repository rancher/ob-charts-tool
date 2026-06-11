package image

import (
	"testing"
)

func TestExtractImages(t *testing.T) {
	tests := []struct {
		name       string
		valuesData string
		defaultTag string
		wantImages []Image
		wantErr    bool
	}{
		{
			name: "single image with all fields",
			valuesData: `
image:
  registry: docker.io
  repository: rancher/example
  tag: v1.0.0
`,
			defaultTag: "",
			wantImages: []Image{
				{Registry: "docker.io", Repository: "rancher/example", Tag: "v1.0.0"},
			},
			wantErr: false,
		},
		{
			name: "image without tag uses defaultTag",
			valuesData: `
image:
  registry: docker.io
  repository: rancher/example
  tag: ""
`,
			defaultTag: "v2.0.0",
			wantImages: []Image{
				{Registry: "docker.io", Repository: "rancher/example", Tag: "v2.0.0"},
			},
			wantErr: false,
		},
		{
			name: "image without tag and no defaultTag",
			valuesData: `
image:
  registry: docker.io
  repository: rancher/example
`,
			defaultTag: "",
			wantImages: []Image{
				{Registry: "docker.io", Repository: "rancher/example", Tag: ""},
			},
			wantErr: false,
		},
		{
			name: "multiple images",
			valuesData: `
image:
  registry: docker.io
  repository: rancher/app
  tag: v1.0.0
kubeRBACProxy:
  image:
    registry: quay.io
    repository: brancz/kube-rbac-proxy
    tag: v0.14.0
`,
			defaultTag: "",
			wantImages: []Image{
				{Registry: "docker.io", Repository: "rancher/app", Tag: "v1.0.0"},
				{Registry: "quay.io", Repository: "brancz/kube-rbac-proxy", Tag: "v0.14.0"},
			},
			wantErr: false,
		},
		{
			name: "nested images",
			valuesData: `
prometheus:
  image:
    registry: quay.io
    repository: prometheus/prometheus
    tag: v2.40.0
  serviceAccount:
    create: true
alertmanager:
  image:
    registry: quay.io
    repository: prometheus/alertmanager
    tag: v0.25.0
`,
			defaultTag: "",
			wantImages: []Image{
				{Registry: "quay.io", Repository: "prometheus/prometheus", Tag: "v2.40.0"},
				{Registry: "quay.io", Repository: "prometheus/alertmanager", Tag: "v0.25.0"},
			},
			wantErr: false,
		},
		{
			name: "case insensitive image key",
			valuesData: `
Image:
  registry: docker.io
  repository: rancher/test
  tag: latest
`,
			defaultTag: "",
			wantImages: []Image{
				{Registry: "docker.io", Repository: "rancher/test", Tag: "latest"},
			},
			wantErr: false,
		},
		{
			name: "prefixed image key",
			valuesData: `
initImage:
  registry: docker.io
  repository: rancher/init
  tag: v1.0.0
`,
			defaultTag: "",
			wantImages: []Image{
				{Registry: "docker.io", Repository: "rancher/init", Tag: "v1.0.0"},
			},
			wantErr: false,
		},
		{
			name:       "empty values",
			valuesData: `{}`,
			defaultTag: "",
			wantImages: nil,
			wantErr:    false,
		},
		{
			name:       "invalid yaml",
			valuesData: `invalid: yaml: [[[`,
			defaultTag: "",
			wantImages: nil,
			wantErr:    true,
		},
		{
			name: "image with only repository",
			valuesData: `
image:
  repository: rancher/simple
`,
			defaultTag: "v1.0.0",
			wantImages: []Image{
				{Registry: "", Repository: "rancher/simple", Tag: "v1.0.0"},
			},
			wantErr: false,
		},
		{
			name: "multiple images with same values but different keys",
			valuesData: `
image:
  repository: rancher/app
  tag: v1.0.0
sidecarImage:
  repository: rancher/sidecar
  tag: v1.0.0
`,
			defaultTag: "",
			wantImages: []Image{
				{Registry: "", Repository: "rancher/app", Tag: "v1.0.0"},
				{Registry: "", Repository: "rancher/sidecar", Tag: "v1.0.0"},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotSet, err := ExtractImages([]byte(tt.valuesData), tt.defaultTag)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExtractImages() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}

			got := gotSet.Values()
			if len(got) != len(tt.wantImages) {
				t.Errorf("ExtractImages() returned %d images, want %d. Got: %+v", len(got), len(tt.wantImages), got)
				return
			}

			// Convert to map for easier comparison (order doesn't matter in a set)
			gotMap := make(map[Image]bool)
			for _, img := range got {
				gotMap[img] = true
			}

			for _, wantImg := range tt.wantImages {
				if !gotMap[wantImg] {
					t.Errorf("ExtractImages() missing expected image %+v. Got: %+v", wantImg, got)
				}
			}
		})
	}
}

func TestExtractImages_EmptyBytes(t *testing.T) {
	images, err := ExtractImages([]byte{}, "")
	if err != nil {
		t.Errorf("ExtractImages() with empty bytes should not error, got: %v", err)
	}
	if images.Size() != 0 {
		t.Errorf("ExtractImages() with empty bytes should return empty set, got %d items", images.Size())
	}
}

func TestExtractImagesFromTemplates(t *testing.T) {
	tests := []struct {
		name           string
		renderedChart  string
		wantContains   []string
		wantNotContain []string
	}{
		{
			name: "simple image reference",
			renderedChart: `
apiVersion: v1
kind: Pod
spec:
  containers:
  - name: app
    image: rancher/app:v1.0.0
`,
			wantContains:   []string{"rancher/app:v1.0.0"},
			wantNotContain: []string{},
		},
		{
			name: "docker.io prefix",
			renderedChart: `
spec:
  containers:
  - image: docker.io/rancher/app:v1.0.0
`,
			wantContains:   []string{"rancher/app:v1.0.0"},
			wantNotContain: []string{},
		},
		{
			name: "exclude registry: lines",
			renderedChart: `
registry: docker.io
spec:
  image: docker.io/rancher/app:v1.0.0
`,
			wantContains:   []string{"rancher/app:v1.0.0"},
			wantNotContain: []string{"registry:"},
		},
		{
			name: "docker.io in env value",
			renderedChart: `
env:
- name: IMAGE
  value: VALUE=docker.io/rancher/test:latest
`,
			wantContains:   []string{"rancher/test:latest"},
			wantNotContain: []string{"VALUE="},
		},
		{
			name: "quoted image",
			renderedChart: `
spec:
  image: "rancher/quoted:v1.0.0"
`,
			wantContains:   []string{"rancher/quoted:v1.0.0"},
			wantNotContain: []string{`"`},
		},
		{
			name: "multiple images",
			renderedChart: `
spec:
  containers:
  - image: rancher/app:v1.0.0
  - image: rancher/sidecar:v2.0.0
  - image: docker.io/rancher/init:latest
`,
			wantContains: []string{
				"rancher/app:v1.0.0",
				"rancher/sidecar:v2.0.0",
				"rancher/init:latest",
			},
			wantNotContain: []string{},
		},
		{
			name:           "empty template",
			renderedChart:  "",
			wantContains:   []string{},
			wantNotContain: []string{},
		},
		{
			name: "template variables excluded",
			renderedChart: `
spec:
  image: {{ .Values.image.repository }}:{{ .Values.image.tag }}
`,
			wantContains:   []string{},
			wantNotContain: []string{"{{"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractImagesFromTemplates(tt.renderedChart)
			gotValues := got.Values()

			for _, want := range tt.wantContains {
				found := false
				for _, val := range gotValues {
					if val == want {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("ExtractImagesFromTemplates() missing expected value %q. Got: %v", want, gotValues)
				}
			}

			for _, notWant := range tt.wantNotContain {
				for _, val := range gotValues {
					if val == notWant {
						t.Errorf("ExtractImagesFromTemplates() contains unexpected value %q. Got: %v", notWant, gotValues)
					}
				}
			}
		})
	}
}

func TestExtractImagesFromTemplates_SetOperations(t *testing.T) {
	// Test that duplicate images are deduplicated (set behavior)
	renderedChart := `
spec:
  containers:
  - image: rancher/app:v1.0.0
  - image: rancher/app:v1.0.0
  - image: rancher/app:v1.0.0
`
	got := ExtractImagesFromTemplates(renderedChart)

	// Count how many times the image appears
	count := 0
	for _, val := range got.Values() {
		if val == "rancher/app:v1.0.0" {
			count++
		}
	}

	if count != 1 {
		t.Errorf("ExtractImagesFromTemplates() should deduplicate, got %d instances of the same image", count)
	}
}
