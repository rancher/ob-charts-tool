package branchverifycheck

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// =============================================================================
// Helpers
// =============================================================================

// writeFile writes content to path, creating all parent directories.
func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%s): %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(%s): %v", path, err)
	}
}

// makePackageInfo returns a PackageInfo with FullPath = name+"/"+versionDir.
func makePackageInfo(name, versionDir string) PackageInfo {
	return PackageInfo{
		FullPath:   name + "/" + versionDir,
		Name:       name,
		VersionDir: versionDir,
	}
}

// packageYAMLContent returns a minimal package.yaml string for the given version.
func packageYAMLContent(version string) string {
	return fmt.Sprintf("version: %s\n", version)
}

// setupPackage writes package.yaml at the correct path for the given package.
// rancher-monitoring uses: packages/<name>/<ver>/<name>/package.yaml
// all others use:          packages/<name>/<ver>/package.yaml
func setupPackage(t *testing.T, repoPath string, pkg PackageInfo, version string) {
	t.Helper()
	path := getPackageYAMLPath(repoPath, pkg)
	writeFile(t, path, packageYAMLContent(version))
}

// setupBuiltChart creates the charts/<pkgName>/<version>/ directory.
func setupBuiltChart(t *testing.T, repoPath, pkgName, version string) {
	t.Helper()
	dir := filepath.Join(repoPath, "charts", pkgName, version)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("MkdirAll(%s): %v", dir, err)
	}
	// Write a placeholder file so the directory is non-empty.
	writeFile(t, filepath.Join(dir, ".keep"), "")
}

// setupSubchart creates Chart.yaml and values.yaml for a subchart inside chartsDir.
func setupSubchart(t *testing.T, chartsDir, subchartName, appVersion, valuesContent string) {
	t.Helper()
	subDir := filepath.Join(chartsDir, subchartName)
	writeFile(t, filepath.Join(subDir, "Chart.yaml"), fmt.Sprintf("appVersion: %s\n", appVersion))
	writeFile(t, filepath.Join(subDir, "values.yaml"), valuesContent)
}

// makeCommittedGitRepo creates a git repo in a temp dir with one initial commit.
// Returns the directory path and the open repository.
// An initial commit is required so that HEAD points to a real branch reference.
func makeCommittedGitRepo(t *testing.T) (string, *git.Repository) {
	t.Helper()
	dir := t.TempDir()
	repo, err := git.PlainInit(dir, false)
	if err != nil {
		t.Fatalf("git.PlainInit: %v", err)
	}

	wt, err := repo.Worktree()
	if err != nil {
		t.Fatalf("Worktree: %v", err)
	}

	// Create a dummy file and commit it so HEAD is a proper branch ref.
	writeFile(t, filepath.Join(dir, "README.md"), "# test\n")
	if _, err := wt.Add("README.md"); err != nil {
		t.Fatalf("git add: %v", err)
	}

	sig := &object.Signature{Name: "Test", Email: "test@test.com", When: time.Now()}
	if _, err := wt.Commit("initial commit", &git.CommitOptions{Author: sig}); err != nil {
		t.Fatalf("git commit: %v", err)
	}

	return dir, repo
}

// addGitRemote adds a remote with the given URL to the repository.
func addGitRemote(t *testing.T, repo *git.Repository, name, url string) {
	t.Helper()
	_, err := repo.CreateRemote(&config.RemoteConfig{Name: name, URLs: []string{url}})
	if err != nil {
		t.Fatalf("CreateRemote(%s): %v", name, err)
	}
}

// setBranch sets the current HEAD to a new branch with the given name,
// pointing to the same commit as the current HEAD.
func setBranch(t *testing.T, repo *git.Repository, branchName string) {
	t.Helper()
	headRef, err := repo.Head()
	if err != nil {
		t.Fatalf("Head(): %v", err)
	}
	// Create the branch ref
	branchRef := plumbing.NewHashReference(
		plumbing.NewBranchReferenceName(branchName),
		headRef.Hash(),
	)
	if err := repo.Storer.SetReference(branchRef); err != nil {
		t.Fatalf("SetReference branch: %v", err)
	}
	// Point HEAD at the new branch
	symRef := plumbing.NewSymbolicReference(plumbing.HEAD, plumbing.NewBranchReferenceName(branchName))
	if err := repo.Storer.SetReference(symRef); err != nil {
		t.Fatalf("SetReference HEAD: %v", err)
	}
}

// =============================================================================
// Pure logic tests
// =============================================================================

func TestExtractRancherRelease(t *testing.T) {
	cases := []struct {
		input       string
		wantRelease int
		wantErr     bool
	}{
		{"1.0.0-rancher.1", 1, false},
		{"1.0.0-rancher.3", 3, false},
		{"1.0.0-rancher.10", 10, false},
		{"77.0.0+up12.0.0-rancher.2", 2, false},
		{"1.0.0", 0, true},
		{"1.0.0-rc.1", 0, true},
		{"", 0, true},
		{"rancher.3", 0, true}, // no dash before "rancher"
	}
	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			got, err := extractRancherRelease(tc.input)
			if (err != nil) != tc.wantErr {
				t.Errorf("extractRancherRelease(%q) error = %v, wantErr %v", tc.input, err, tc.wantErr)
			}
			if !tc.wantErr && got != tc.wantRelease {
				t.Errorf("extractRancherRelease(%q) = %d, want %d", tc.input, got, tc.wantRelease)
			}
		})
	}
}

func TestIsImageDefinition(t *testing.T) {
	cases := []struct {
		name string
		m    map[string]any
		want bool
	}{
		{"repository and tag", map[string]any{"repository": "rancher/foo", "tag": "v1"}, true},
		{"repository, tag, registry", map[string]any{"repository": "rancher/foo", "tag": "v1", "registry": ""}, true},
		{"repository only (no tag)", map[string]any{"repository": "rancher/foo"}, false},
		{"tag only (no repository)", map[string]any{"tag": "v1"}, false},
		{"empty map", map[string]any{}, false},
		{"registry and tag only", map[string]any{"registry": "", "tag": "v1"}, false},
		{"extra keys OK", map[string]any{"repository": "rancher/foo", "tag": "v1", "pullPolicy": "IfNotPresent"}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Convert map[string]any to map[string]interface{} for the function signature
			m := make(map[string]interface{}, len(tc.m))
			for k, v := range tc.m {
				m[k] = v
			}
			if got := isImageDefinition(m); got != tc.want {
				t.Errorf("isImageDefinition() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestValidateImageDefinition(t *testing.T) {
	cases := []struct {
		name       string
		m          map[string]any
		wantIssues int
		wantSubstr string // substring expected in the issues (if wantIssues > 0)
	}{
		{
			name:       "valid: empty registry + rancher/ repo",
			m:          map[string]any{"repository": "rancher/foo", "tag": "v1", "registry": ""},
			wantIssues: 0,
		},
		{
			name:       "valid: no registry key",
			m:          map[string]any{"repository": "rancher/foo", "tag": "v1"},
			wantIssues: 0,
		},
		{
			name:       "invalid: non-empty registry",
			m:          map[string]any{"repository": "rancher/foo", "tag": "v1", "registry": "docker.io"},
			wantIssues: 1,
			wantSubstr: "registry=docker.io",
		},
		{
			name:       "invalid: repository not rancher/",
			m:          map[string]any{"repository": "notrancher/foo", "tag": "v1", "registry": ""},
			wantIssues: 1,
			wantSubstr: "repository=notrancher/foo",
		},
		{
			name:       "both invalid",
			m:          map[string]any{"repository": "notrancher/foo", "tag": "v1", "registry": "quay.io"},
			wantIssues: 2,
		},
		{
			name:       "no repository key: returns early",
			m:          map[string]any{"tag": "v1"},
			wantIssues: 0,
		},
		{
			name:       "non-string repository: returns early",
			m:          map[string]any{"repository": 123, "tag": "v1"},
			wantIssues: 0,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := make(map[string]interface{}, len(tc.m))
			for k, v := range tc.m {
				m[k] = v
			}
			var invalidImages []InvalidImage
			validateImageDefinition(m, "test.path", &invalidImages)
			if len(invalidImages) == 0 {
				if tc.wantIssues > 0 {
					t.Errorf("validateImageDefinition: got 0 invalid images, want %d", tc.wantIssues)
				}
				return
			}
			totalIssues := 0
			for _, img := range invalidImages {
				totalIssues += len(img.Issues)
			}
			if totalIssues != tc.wantIssues {
				t.Errorf("validateImageDefinition: got %d issues, want %d: %+v", totalIssues, tc.wantIssues, invalidImages)
			}
			if tc.wantSubstr != "" {
				found := false
				for _, img := range invalidImages {
					for _, issue := range img.Issues {
						if strings.Contains(issue, tc.wantSubstr) {
							found = true
						}
					}
				}
				if !found {
					t.Errorf("validateImageDefinition: no issue contains %q: %+v", tc.wantSubstr, invalidImages)
				}
			}
		})
	}
}

func TestFindInvalidImages(t *testing.T) {
	cases := []struct {
		name             string
		data             any
		wantInvalidCount int
		wantPath         string // path of first invalid image, if any
	}{
		{
			name: "flat valid image",
			data: map[string]any{
				"image": map[string]any{"repository": "rancher/foo", "tag": "v1", "registry": ""},
			},
			wantInvalidCount: 0,
		},
		{
			name: "flat invalid image (bad registry)",
			data: map[string]any{
				"image": map[string]any{"repository": "rancher/foo", "tag": "v1", "registry": "docker.io"},
			},
			wantInvalidCount: 1,
			wantPath:         "image",
		},
		{
			name: "nested two levels",
			data: map[string]any{
				"global": map[string]any{
					"image": map[string]any{"repository": "bad/foo", "tag": "v1"},
				},
			},
			wantInvalidCount: 1,
			wantPath:         "global.image",
		},
		{
			name: "array of maps: one invalid",
			data: map[string]any{
				"images": []any{
					map[string]any{"repository": "rancher/a", "tag": "v1"},
					map[string]any{"repository": "bad/b", "tag": "v2"},
				},
			},
			wantInvalidCount: 1,
		},
		{
			name: "non-image map: not flagged",
			data: map[string]any{
				"config": map[string]any{"replicas": 3, "name": "foo"},
			},
			wantInvalidCount: 0,
		},
		{
			name: "image map stops recursion into children",
			data: map[string]any{
				"img": map[string]any{
					"repository": "rancher/foo",
					"tag":        "v1",
					// nested inside an image map — should NOT be checked separately
					"nested": map[string]any{"repository": "bad/x", "tag": "v2"},
				},
			},
			wantInvalidCount: 0,
		},
		{
			name: "path propagated correctly",
			data: map[string]any{
				"outer": map[string]any{
					"img": map[string]any{"repository": "bad/x", "tag": "v1"},
				},
			},
			wantInvalidCount: 1,
			wantPath:         "outer.img",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var invalidImages []InvalidImage
			findInvalidImages(tc.data, "", &invalidImages)
			if len(invalidImages) != tc.wantInvalidCount {
				t.Errorf("findInvalidImages: got %d invalid images, want %d: %+v", len(invalidImages), tc.wantInvalidCount, invalidImages)
				return
			}
			if tc.wantPath != "" && len(invalidImages) > 0 {
				if invalidImages[0].Path != tc.wantPath {
					t.Errorf("findInvalidImages: path = %q, want %q", invalidImages[0].Path, tc.wantPath)
				}
			}
		})
	}
}

func TestGetPackageYAMLPath(t *testing.T) {
	cases := []struct {
		pkgName    string
		versionDir string
		wantSuffix string // use filepath.Join to build
	}{
		{
			"rancher-monitoring", "77.9",
			filepath.Join("packages", "rancher-monitoring", "77.9", "rancher-monitoring", "package.yaml"),
		},
		{
			"rancher-logging", "4.1",
			filepath.Join("packages", "rancher-logging", "4.1", "package.yaml"),
		},
		{
			"rancher-project-monitoring", "0.3",
			filepath.Join("packages", "rancher-project-monitoring", "0.3", "package.yaml"),
		},
	}
	for _, tc := range cases {
		t.Run(tc.pkgName, func(t *testing.T) {
			pkg := makePackageInfo(tc.pkgName, tc.versionDir)
			got := getPackageYAMLPath("/repo", pkg)
			want := filepath.Join("/repo", tc.wantSuffix)
			if got != want {
				t.Errorf("getPackageYAMLPath() = %q, want %q", got, want)
			}
		})
	}
}

// =============================================================================
// Filesystem-dependent tests
// =============================================================================

func TestFindValuesYAMLFiles(t *testing.T) {
	cases := []struct {
		name      string
		files     []string // relative paths to create
		wantCount int
	}{
		{"empty dir", nil, 0},
		{"single values.yaml", []string{"values.yaml"}, 1},
		{"values.yml variant", []string{"values.yml"}, 1},
		{"both variants", []string{"values.yaml", "values.yml"}, 2},
		{"nested file found", []string{"charts/sub/values.yaml"}, 1},
		{"non-values files ignored", []string{"Chart.yaml", "values-override.yaml"}, 0},
		{"deeply nested", []string{"a/b/c/values.yaml"}, 1},
		{"mix: one valid, one invalid", []string{"values.yaml", "Chart.yaml"}, 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			for _, f := range tc.files {
				writeFile(t, filepath.Join(dir, f), "")
			}
			got, err := findValuesYAMLFiles(dir)
			if err != nil {
				t.Fatalf("findValuesYAMLFiles: %v", err)
			}
			if len(got) != tc.wantCount {
				t.Errorf("findValuesYAMLFiles: got %d files, want %d: %v", len(got), tc.wantCount, got)
			}
		})
	}
}

func TestCheckChartBuilt(t *testing.T) {
	cases := []struct {
		name           string
		pkgName        string
		versionDir     string
		packageVersion string
		builtVersions  []string
		wantPassed     bool
	}{
		{
			name:    "version exists in charts/",
			pkgName: "rancher-logging", versionDir: "4.1",
			packageVersion: "1.0.0", builtVersions: []string{"1.0.0"},
			wantPassed: true,
		},
		{
			name:    "version missing from charts/",
			pkgName: "rancher-logging", versionDir: "4.1",
			packageVersion: "1.0.0", builtVersions: []string{"0.9.0"},
			wantPassed: false,
		},
		{
			name:    "charts/ dir exists but is empty",
			pkgName: "rancher-logging", versionDir: "4.1",
			packageVersion: "1.0.0", builtVersions: nil,
			wantPassed: false,
		},
		{
			name:    "charts/ dir does not exist",
			pkgName: "rancher-logging", versionDir: "4.1",
			packageVersion: "1.0.0", builtVersions: nil,
			wantPassed: false,
		},
		{
			name:    "rancher-monitoring uses nested package.yaml path",
			pkgName: "rancher-monitoring", versionDir: "77.9",
			packageVersion: "1.0.0-rancher.1", builtVersions: []string{"1.0.0-rancher.1"},
			wantPassed: true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repoPath := t.TempDir()
			pkg := makePackageInfo(tc.pkgName, tc.versionDir)
			setupPackage(t, repoPath, pkg, tc.packageVersion)
			// Create charts dir only if we have built versions to add (empty charts/ test uses MkdirAll with no subdirs)
			if tc.name == "charts/ dir exists but is empty" {
				if err := os.MkdirAll(filepath.Join(repoPath, "charts", tc.pkgName), 0o755); err != nil {
					t.Fatalf("MkdirAll: %v", err)
				}
			}
			for _, v := range tc.builtVersions {
				setupBuiltChart(t, repoPath, tc.pkgName, v)
			}
			result := CheckChartBuilt(repoPath, pkg)
			if result.Passed != tc.wantPassed {
				t.Errorf("CheckChartBuilt.Passed = %v, want %v: %s", result.Passed, tc.wantPassed, result.Message)
			}
		})
	}

	t.Run("package.yaml missing", func(t *testing.T) {
		repoPath := t.TempDir()
		pkg := makePackageInfo("rancher-logging", "4.1")
		// No package.yaml written
		result := CheckChartBuilt(repoPath, pkg)
		if result.Passed {
			t.Errorf("CheckChartBuilt should fail when package.yaml is missing")
		}
	})
}

func TestCheckSequentialVersion(t *testing.T) {
	cases := []struct {
		name           string
		pkgName        string
		versionDir     string
		currentVersion string
		existingCharts []string // chart versions to create in charts/
		wantPassed     bool
	}{
		{
			name:    "rancher suffix: n with n-1 exists",
			pkgName: "rancher-logging", versionDir: "4.1",
			currentVersion: "1.0.0-rancher.3",
			existingCharts: []string{"1.0.0-rancher.2", "1.0.0-rancher.3"},
			wantPassed:     true,
		},
		{
			name:    "rancher suffix: n missing n-1",
			pkgName: "rancher-logging", versionDir: "4.1",
			currentVersion: "1.0.0-rancher.3",
			existingCharts: []string{"1.0.0-rancher.1", "1.0.0-rancher.3"},
			wantPassed:     false,
		},
		{
			name:    "rancher suffix: .1 is always valid (first rancher release)",
			pkgName: "rancher-logging", versionDir: "4.1",
			currentVersion: "1.0.0-rancher.1",
			existingCharts: []string{"0.9.0-rancher.2", "1.0.0-rancher.1"},
			wantPassed:     true,
		},
		{
			name:    "rancher suffix: multi-digit release",
			pkgName: "rancher-logging", versionDir: "4.1",
			currentVersion: "1.0.0-rancher.10",
			existingCharts: []string{"1.0.0-rancher.9", "1.0.0-rancher.10"},
			wantPassed:     true,
		},
		{
			name:    "semver: lower version exists",
			pkgName: "rancher-project-monitoring", versionDir: "0.3",
			currentVersion: "1.2.3",
			existingCharts: []string{"1.2.2", "1.2.3"},
			wantPassed:     true,
		},
		{
			name:    "semver: any lower version is valid",
			pkgName: "rancher-project-monitoring", versionDir: "0.3",
			currentVersion: "1.2.3",
			existingCharts: []string{"1.1.0", "1.2.3"},
			wantPassed:     true,
		},
		{
			name:    "semver: no lower version exists",
			pkgName: "rancher-project-monitoring", versionDir: "0.3",
			currentVersion: "1.2.3",
			existingCharts: []string{"1.2.4", "1.2.3"},
			wantPassed:     false,
		},
		{
			name:    "semver: only higher versions",
			pkgName: "rancher-project-monitoring", versionDir: "0.3",
			currentVersion: "1.2.3",
			existingCharts: []string{"2.0.0", "1.5.0", "1.2.3"},
			wantPassed:     false,
		},
		{
			name:    "first version ever (no charts/ dir at all)",
			pkgName: "rancher-logging", versionDir: "4.1",
			currentVersion: "1.0.0",
			existingCharts: nil,
			wantPassed:     true,
		},
		{
			name:    "only current version in charts/ (no others)",
			pkgName: "rancher-logging", versionDir: "4.1",
			currentVersion: "1.0.0",
			existingCharts: []string{"1.0.0"},
			wantPassed:     true,
		},
		{
			name:    "rancher-monitoring uses nested package.yaml path",
			pkgName: "rancher-monitoring", versionDir: "77.9",
			currentVersion: "1.0.0-rancher.2",
			existingCharts: []string{"1.0.0-rancher.1", "1.0.0-rancher.2"},
			wantPassed:     true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repoPath := t.TempDir()
			pkg := makePackageInfo(tc.pkgName, tc.versionDir)
			setupPackage(t, repoPath, pkg, tc.currentVersion)
			for _, v := range tc.existingCharts {
				setupBuiltChart(t, repoPath, tc.pkgName, v)
			}
			result := CheckSequentialVersion(repoPath, pkg)
			if result.Passed != tc.wantPassed {
				t.Errorf("CheckSequentialVersion.Passed = %v, want %v: %s", result.Passed, tc.wantPassed, result.Message)
			}
		})
	}

	t.Run("invalid version format in package.yaml", func(t *testing.T) {
		repoPath := t.TempDir()
		pkg := makePackageInfo("rancher-logging", "4.1")
		setupPackage(t, repoPath, pkg, "not-a-version")
		setupBuiltChart(t, repoPath, "rancher-logging", "1.0.0")
		result := CheckSequentialVersion(repoPath, pkg)
		if result.Passed {
			t.Errorf("CheckSequentialVersion should fail for invalid version format")
		}
	})
}

func TestCheckPackageImages(t *testing.T) {
	const currentVersion = "1.0.0"
	pkgName := "rancher-logging"
	versionDir := "4.1"

	setup := func(t *testing.T) (string, PackageInfo) {
		t.Helper()
		repoPath := t.TempDir()
		pkg := makePackageInfo(pkgName, versionDir)
		setupPackage(t, repoPath, pkg, currentVersion)
		setupBuiltChart(t, repoPath, pkgName, currentVersion)
		return repoPath, pkg
	}

	chartDir := func(repoPath string) string {
		return filepath.Join(repoPath, "charts", pkgName, currentVersion)
	}

	cases := []struct {
		name        string
		valuesYAML  string
		wantPassed  bool
		wantInvalid int
	}{
		{
			name: "valid: empty registry and rancher/ repo",
			valuesYAML: `image:
  registry: ""
  repository: rancher/foo
  tag: v1.0
`,
			wantPassed: true,
		},
		{
			name: "invalid: non-empty registry",
			valuesYAML: `image:
  registry: docker.io
  repository: rancher/foo
  tag: v1.0
`,
			wantPassed:  false,
			wantInvalid: 1,
		},
		{
			name: "invalid: repository not rancher/",
			valuesYAML: `image:
  registry: ""
  repository: notrancher/foo
  tag: v1.0
`,
			wantPassed:  false,
			wantInvalid: 1,
		},
		{
			name: "both invalid on same image",
			valuesYAML: `image:
  registry: quay.io
  repository: notrancher/foo
  tag: v1.0
`,
			wantPassed:  false,
			wantInvalid: 1, // 1 invalid image (with 2 issues)
		},
		{
			name:       "no repository+tag map: passes",
			valuesYAML: "config:\n  replicas: 3\n",
			wantPassed: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repoPath, pkg := setup(t)
			writeFile(t, filepath.Join(chartDir(repoPath), "values.yaml"), tc.valuesYAML)
			result := CheckPackageImages(repoPath, pkg)
			if result.Passed != tc.wantPassed {
				t.Errorf("CheckPackageImages.Passed = %v, want %v: %s", result.Passed, tc.wantPassed, result.Message)
			}
			if !tc.wantPassed && tc.wantInvalid > 0 && result.Details != nil {
				details, ok := result.Details.(*ImageCheckDetails)
				if !ok {
					t.Errorf("Details is not *ImageCheckDetails")
				} else if len(details.InvalidImages) != tc.wantInvalid {
					t.Errorf("InvalidImages count = %d, want %d", len(details.InvalidImages), tc.wantInvalid)
				}
			}
		})
	}

	t.Run("no values.yaml files: passes", func(t *testing.T) {
		repoPath, pkg := setup(t)
		// No values.yaml written — just the .keep file from setupBuiltChart
		result := CheckPackageImages(repoPath, pkg)
		if !result.Passed {
			t.Errorf("CheckPackageImages should pass when no values.yaml files exist: %s", result.Message)
		}
	})

	t.Run("multiple values files: one invalid", func(t *testing.T) {
		repoPath, pkg := setup(t)
		writeFile(t, filepath.Join(chartDir(repoPath), "values.yaml"), `image:
  registry: ""
  repository: rancher/good
  tag: v1
`)
		writeFile(t, filepath.Join(chartDir(repoPath), "charts", "sub", "values.yaml"), `image:
  registry: docker.io
  repository: rancher/bad
  tag: v1
`)
		result := CheckPackageImages(repoPath, pkg)
		if result.Passed {
			t.Errorf("CheckPackageImages should fail when one of multiple values files has invalid image")
		}
	})
}

func TestCheckSubchartAppVersionTags(t *testing.T) {
	const version = "77.0.0"
	const versionDir = "77.0"
	const monPkgName = "rancher-monitoring"

	setupMon := func(t *testing.T) (string, PackageInfo) {
		t.Helper()
		repoPath := t.TempDir()
		pkg := makePackageInfo(monPkgName, versionDir)
		setupPackage(t, repoPath, pkg, version)
		setupBuiltChart(t, repoPath, monPkgName, version)
		return repoPath, pkg
	}

	subchartsDir := func(repoPath string) string {
		return filepath.Join(repoPath, "charts", monPkgName, version, "charts")
	}

	t.Run("not rancher-monitoring: skipped", func(t *testing.T) {
		repoPath := t.TempDir()
		pkg := makePackageInfo("rancher-logging", "4.1")
		setupPackage(t, repoPath, pkg, "1.0.0")
		result := CheckSubchartAppVersionTags(repoPath, pkg)
		if !result.Passed {
			t.Errorf("should pass (skipped) for non-rancher-monitoring: %s", result.Message)
		}
	})

	t.Run("no charts/ subdir: skipped", func(t *testing.T) {
		repoPath, pkg := setupMon(t)
		// charts/<version>/charts/ does not exist
		result := CheckSubchartAppVersionTags(repoPath, pkg)
		if !result.Passed {
			t.Errorf("should pass (skipped) when charts/ subdir missing: %s", result.Message)
		}
	})

	t.Run("grafana: matching tag", func(t *testing.T) {
		repoPath, pkg := setupMon(t)
		setupSubchart(t, subchartsDir(repoPath), "grafana", "10.0.0",
			"image:\n  tag: \"10.0.0\"\n")
		result := CheckSubchartAppVersionTags(repoPath, pkg)
		if !result.Passed {
			t.Errorf("should pass for matching grafana tag: %s", result.Message)
		}
	})

	t.Run("grafana: mismatched tag", func(t *testing.T) {
		repoPath, pkg := setupMon(t)
		setupSubchart(t, subchartsDir(repoPath), "grafana", "10.0.0",
			"image:\n  tag: \"9.0.0\"\n")
		result := CheckSubchartAppVersionTags(repoPath, pkg)
		if result.Passed {
			t.Errorf("should fail for mismatched grafana tag")
		}
		details, ok := result.Details.(*SubchartTagCheckDetails)
		if !ok {
			t.Fatalf("Details is not *SubchartTagCheckDetails")
		}
		if len(details.Mismatches) != 1 {
			t.Errorf("want 1 mismatch, got %d", len(details.Mismatches))
		}
	})

	t.Run("kube-state-metrics: correct v prefix on both rules", func(t *testing.T) {
		repoPath, pkg := setupMon(t)
		setupSubchart(t, subchartsDir(repoPath), "kube-state-metrics", "2.10.0",
			"image:\n  tag: \"v2.10.0\"\nkubeRBACProxy:\n  image:\n    tag: \"v2.10.0\"\n")
		result := CheckSubchartAppVersionTags(repoPath, pkg)
		if !result.Passed {
			t.Errorf("should pass for kube-state-metrics with correct v prefix: %s", result.Message)
		}
	})

	t.Run("kube-state-metrics: missing v prefix", func(t *testing.T) {
		repoPath, pkg := setupMon(t)
		setupSubchart(t, subchartsDir(repoPath), "kube-state-metrics", "2.10.0",
			"image:\n  tag: \"2.10.0\"\nkubeRBACProxy:\n  image:\n    tag: \"v2.10.0\"\n")
		result := CheckSubchartAppVersionTags(repoPath, pkg)
		if result.Passed {
			t.Errorf("should fail when kube-state-metrics image.tag is missing v prefix")
		}
	})

	t.Run("unknown subchart not in SubchartsToCheck: ignored", func(t *testing.T) {
		repoPath, pkg := setupMon(t)
		setupSubchart(t, subchartsDir(repoPath), "some-unknown-chart", "1.0.0",
			"image:\n  tag: \"completely-wrong\"\n")
		result := CheckSubchartAppVersionTags(repoPath, pkg)
		if !result.Passed {
			t.Errorf("should pass when subchart is not in SubchartsToCheck: %s", result.Message)
		}
	})

	t.Run("rancher-prefixed subdir: NormalizeName strips prefix", func(t *testing.T) {
		repoPath, pkg := setupMon(t)
		// "rancher-grafana" normalizes to "grafana" via NormalizeName
		setupSubchart(t, subchartsDir(repoPath), "rancher-grafana", "10.0.0",
			"image:\n  tag: \"10.0.0\"\n")
		result := CheckSubchartAppVersionTags(repoPath, pkg)
		if !result.Passed {
			t.Errorf("should pass for rancher-grafana with matching tag: %s", result.Message)
		}
	})

	t.Run("Chart.yaml missing: subchart skipped", func(t *testing.T) {
		repoPath, pkg := setupMon(t)
		// Create the directory but only values.yaml, no Chart.yaml
		subDir := filepath.Join(subchartsDir(repoPath), "grafana")
		writeFile(t, filepath.Join(subDir, "values.yaml"), "image:\n  tag: \"wrong\"\n")
		result := CheckSubchartAppVersionTags(repoPath, pkg)
		if !result.Passed {
			t.Errorf("should pass when Chart.yaml is missing (subchart skipped): %s", result.Message)
		}
	})

	t.Run("values.yaml missing: subchart skipped", func(t *testing.T) {
		repoPath, pkg := setupMon(t)
		subDir := filepath.Join(subchartsDir(repoPath), "grafana")
		writeFile(t, filepath.Join(subDir, "Chart.yaml"), "appVersion: 10.0.0\n")
		// No values.yaml
		result := CheckSubchartAppVersionTags(repoPath, pkg)
		if !result.Passed {
			t.Errorf("should pass when values.yaml is missing (subchart skipped): %s", result.Message)
		}
	})
}

// =============================================================================
// Git-dependent tests
// =============================================================================

func TestCheckIsGitRepo(t *testing.T) {
	t.Run("valid git repo", func(t *testing.T) {
		dir := t.TempDir()
		if _, err := git.PlainInit(dir, false); err != nil {
			t.Fatalf("PlainInit: %v", err)
		}
		result := CheckIsGitRepo(dir)
		if !result.Passed {
			t.Errorf("CheckIsGitRepo should pass for valid git repo: %s", result.Message)
		}
	})

	t.Run("empty dir (not a git repo)", func(t *testing.T) {
		dir := t.TempDir()
		result := CheckIsGitRepo(dir)
		if result.Passed {
			t.Errorf("CheckIsGitRepo should fail for a non-git directory")
		}
	})

	t.Run("non-existent path", func(t *testing.T) {
		result := CheckIsGitRepo("/nonexistent/path/that/does/not/exist")
		if result.Passed {
			t.Errorf("CheckIsGitRepo should fail for non-existent path")
		}
	})
}

func TestCheckHasObTeamChartsRemote(t *testing.T) {
	cases := []struct {
		name       string
		remoteURL  string
		wantPassed bool
	}{
		{"HTTPS with .git suffix", "https://github.com/rancher/ob-team-charts.git", true},
		{"HTTPS without .git suffix", "https://github.com/rancher/ob-team-charts", true},
		{"SSH format", "git@github.com:rancher/ob-team-charts.git", true},
		{"HTTPS uppercase owner (case-insensitive)", "https://github.com/Rancher/ob-team-charts.git", true},
		{"wrong repo name", "https://github.com/rancher/helm-charts.git", false},
		{"wrong owner", "https://github.com/someone/ob-team-charts.git", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, repo := makeCommittedGitRepo(t)
			addGitRemote(t, repo, "origin", tc.remoteURL)
			result := CheckHasObTeamChartsRemote(repo)
			if result.Passed != tc.wantPassed {
				t.Errorf("CheckHasObTeamChartsRemote.Passed = %v, want %v: %s", result.Passed, tc.wantPassed, result.Message)
			}
		})
	}

	t.Run("no remotes", func(t *testing.T) {
		_, repo := makeCommittedGitRepo(t)
		// No remotes added
		result := CheckHasObTeamChartsRemote(repo)
		if result.Passed {
			t.Errorf("CheckHasObTeamChartsRemote should fail when no remotes exist")
		}
	})

	t.Run("multiple remotes: one correct", func(t *testing.T) {
		_, repo := makeCommittedGitRepo(t)
		addGitRemote(t, repo, "wrong", "https://github.com/rancher/helm-charts.git")
		addGitRemote(t, repo, "upstream", "https://github.com/rancher/ob-team-charts.git")
		result := CheckHasObTeamChartsRemote(repo)
		if !result.Passed {
			t.Errorf("CheckHasObTeamChartsRemote should pass when at least one remote is correct: %s", result.Message)
		}
	})
}

func TestCheckOnFeatureBranch(t *testing.T) {
	cases := []struct {
		name           string
		branchName     string
		wantPassed     bool
		wantBranchName string
	}{
		{"feature branch", "my-feature", true, "my-feature"},
		{"main branch fails", "main", false, "main"},
		{"master branch fails", "master", false, "master"},
		{"branch with slash", "feature/my-work", true, "feature/my-work"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, repo := makeCommittedGitRepo(t)
			// After makeCommittedGitRepo, HEAD is on "master".
			// Switch to the desired branch.
			setBranch(t, repo, tc.branchName)

			gotName, result := CheckOnFeatureBranch(repo)
			if result.Passed != tc.wantPassed {
				t.Errorf("CheckOnFeatureBranch.Passed = %v, want %v: %s", result.Passed, tc.wantPassed, result.Message)
			}
			if gotName != tc.wantBranchName {
				t.Errorf("branch name = %q, want %q", gotName, tc.wantBranchName)
			}
		})
	}
}

func TestCheckRepoClean(t *testing.T) {
	t.Run("clean repo after commit", func(t *testing.T) {
		_, repo := makeCommittedGitRepo(t)
		result := CheckRepoClean(repo)
		if !result.Passed {
			t.Errorf("CheckRepoClean should pass for clean repo: %s", result.Message)
		}
	})

	t.Run("dirty: untracked file", func(t *testing.T) {
		dir, repo := makeCommittedGitRepo(t)
		writeFile(t, filepath.Join(dir, "untracked.txt"), "new file\n")
		result := CheckRepoClean(repo)
		if result.Passed {
			t.Errorf("CheckRepoClean should fail when untracked file exists")
		}
	})

	t.Run("dirty: staged but uncommitted file", func(t *testing.T) {
		dir, repo := makeCommittedGitRepo(t)
		writeFile(t, filepath.Join(dir, "staged.txt"), "staged content\n")
		wt, err := repo.Worktree()
		if err != nil {
			t.Fatalf("Worktree: %v", err)
		}
		if _, err := wt.Add("staged.txt"); err != nil {
			t.Fatalf("git add: %v", err)
		}
		result := CheckRepoClean(repo)
		if result.Passed {
			t.Errorf("CheckRepoClean should fail when staged changes exist")
		}
	})

	t.Run("dirty: modified tracked file (not staged)", func(t *testing.T) {
		dir, repo := makeCommittedGitRepo(t)
		// README.md was committed in makeCommittedGitRepo — modify it without staging
		writeFile(t, filepath.Join(dir, "README.md"), "# modified content\n")
		result := CheckRepoClean(repo)
		if result.Passed {
			t.Errorf("CheckRepoClean should fail when tracked file is modified but not staged")
		}
	})
}
