package branchverifycheck

import (
	"strings"
	"testing"
)

// cr builds a minimal CheckResult for use in table-driven tests.
func cr(passed, critical bool) CheckResult {
	return CheckResult{Passed: passed, Critical: critical}
}

func TestPackageResult_HasCriticalFailure(t *testing.T) {
	cases := []struct {
		name   string
		checks []CheckResult
		want   bool
	}{
		{"empty checks", nil, false},
		{"single passed critical", []CheckResult{cr(true, true)}, false},
		{"single failed non-critical", []CheckResult{cr(false, false)}, false},
		{"single failed critical", []CheckResult{cr(false, true)}, true},
		{"failed non-critical + passed critical", []CheckResult{cr(false, false), cr(true, true)}, false},
		{"passed non-critical + failed critical", []CheckResult{cr(true, false), cr(false, true)}, true},
		{"multiple non-critical failures only", []CheckResult{cr(false, false), cr(false, false)}, false},
		{"one of three is failed critical", []CheckResult{cr(true, true), cr(false, false), cr(false, true)}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			pr := PackageResult{Checks: tc.checks}
			if got := pr.HasCriticalFailure(); got != tc.want {
				t.Errorf("HasCriticalFailure() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestVerificationResult_HasCriticalFailure(t *testing.T) {
	cases := []struct {
		name          string
		globalChecks  []CheckResult
		packageChecks []CheckResult // added as a single PackageResult
		want          bool
	}{
		{"empty", nil, nil, false},
		{"global failed critical", []CheckResult{cr(false, true)}, nil, true},
		{"global failed non-critical only", []CheckResult{cr(false, false)}, nil, false},
		{"global passed", []CheckResult{cr(true, true)}, nil, false},
		{"package failed critical", nil, []CheckResult{cr(false, true)}, true},
		{"package failed non-critical only", nil, []CheckResult{cr(false, false)}, false},
		{"global passes, package fails critical", []CheckResult{cr(true, true)}, []CheckResult{cr(false, true)}, true},
		{"all pass", []CheckResult{cr(true, true)}, []CheckResult{cr(true, true)}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := &VerificationResult{GlobalChecks: tc.globalChecks}
			if len(tc.packageChecks) > 0 {
				r.PackageResults = []PackageResult{
					{Package: PackageInfo{FullPath: "pkg/1.0"}, Checks: tc.packageChecks},
				}
			}
			if got := r.HasCriticalFailure(); got != tc.want {
				t.Errorf("HasCriticalFailure() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestVerificationResult_CountResults(t *testing.T) {
	cases := []struct {
		name          string
		globalChecks  []CheckResult
		packageChecks []CheckResult
		wantPassed    int
		wantFailed    int
		wantWarnings  int
	}{
		{"empty", nil, nil, 0, 0, 0},
		{"one global pass", []CheckResult{cr(true, true)}, nil, 1, 0, 0},
		{"one global critical fail", []CheckResult{cr(false, true)}, nil, 0, 1, 0},
		{"one global warning (non-critical fail)", []CheckResult{cr(false, false)}, nil, 0, 0, 1},
		{
			name:         "mixed global: 1 pass, 1 fail, 1 warn",
			globalChecks: []CheckResult{cr(true, true), cr(false, true), cr(false, false)},
			wantPassed:   1, wantFailed: 1, wantWarnings: 1,
		},
		{
			name:          "package checks contribute",
			packageChecks: []CheckResult{cr(true, false), cr(false, true), cr(false, false)},
			wantPassed:    1, wantFailed: 1, wantWarnings: 1,
		},
		{
			name:          "global + package combined",
			globalChecks:  []CheckResult{cr(true, true), cr(false, false)},
			packageChecks: []CheckResult{cr(false, true)},
			wantPassed:    1, wantFailed: 1, wantWarnings: 1,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := &VerificationResult{GlobalChecks: tc.globalChecks}
			if len(tc.packageChecks) > 0 {
				r.PackageResults = []PackageResult{
					{Package: PackageInfo{FullPath: "pkg/1.0"}, Checks: tc.packageChecks},
				}
			}
			passed, failed, warnings := r.CountResults()
			if passed != tc.wantPassed || failed != tc.wantFailed || warnings != tc.wantWarnings {
				t.Errorf("CountResults() = (%d, %d, %d), want (%d, %d, %d)",
					passed, failed, warnings, tc.wantPassed, tc.wantFailed, tc.wantWarnings)
			}
		})
	}
}

func TestVerificationResult_GetOrCreatePackageResult(t *testing.T) {
	r := &VerificationResult{}

	pkg1 := PackageInfo{FullPath: "pkg-a/1.0", Name: "pkg-a", VersionDir: "1.0"}
	pkg2 := PackageInfo{FullPath: "pkg-b/2.0", Name: "pkg-b", VersionDir: "2.0"}

	// First call creates entry
	pr1 := r.GetOrCreatePackageResult(pkg1)
	if pr1 == nil {
		t.Fatal("GetOrCreatePackageResult returned nil")
	}
	if len(r.PackageResults) != 1 {
		t.Errorf("len(PackageResults) = %d, want 1", len(r.PackageResults))
	}

	// Same key returns same pointer (no duplicate)
	pr1again := r.GetOrCreatePackageResult(pkg1)
	if pr1again != pr1 {
		t.Errorf("second call with same key returned different pointer")
	}
	if len(r.PackageResults) != 1 {
		t.Errorf("len(PackageResults) = %d, want 1 (no duplicate)", len(r.PackageResults))
	}

	// Different key creates new entry
	pr2 := r.GetOrCreatePackageResult(pkg2)
	if pr2 == nil {
		t.Fatal("GetOrCreatePackageResult returned nil for second package")
	}
	if len(r.PackageResults) != 2 {
		t.Errorf("len(PackageResults) = %d, want 2", len(r.PackageResults))
	}
	if pr2 == pr1 {
		t.Errorf("second package returned same pointer as first")
	}
}

func TestCheckDetails_Format(t *testing.T) {
	t.Run("BuildDiffDetails", func(t *testing.T) {
		d := &BuildDiffDetails{
			ModifiedFiles: []string{"charts/foo/1.0/values.yaml"},
			Diff:          "- old line\n+ new line",
		}
		out := d.Format()
		if !strings.Contains(out, "charts/foo/1.0/values.yaml") {
			t.Errorf("Format() missing modified file: %s", out)
		}
		if !strings.Contains(out, "Diff:") {
			t.Errorf("Format() missing 'Diff:' header: %s", out)
		}
		if !strings.Contains(out, "- old line") {
			t.Errorf("Format() missing diff content: %s", out)
		}
	})

	t.Run("ImageCheckDetails", func(t *testing.T) {
		d := &ImageCheckDetails{
			InvalidImages: []InvalidImage{
				{Path: "image", Issues: []string{"registry=docker.io (expected \"\")"}},
			},
			FilesChecked: 3,
		}
		out := d.Format()
		if !strings.Contains(out, "1") {
			t.Errorf("Format() missing invalid image count: %s", out)
		}
		if !strings.Contains(out, "3") {
			t.Errorf("Format() missing files checked count: %s", out)
		}
		if !strings.Contains(out, "image") {
			t.Errorf("Format() missing image path: %s", out)
		}
		if !strings.Contains(out, "registry=docker.io") {
			t.Errorf("Format() missing issue description: %s", out)
		}
	})

	t.Run("SubchartTagCheckDetails", func(t *testing.T) {
		d := &SubchartTagCheckDetails{
			Mismatches: []SubchartTagMismatch{
				{
					SubchartName:  "grafana",
					ValuesKey:     "image.tag",
					ActualValue:   "9.0.0",
					ExpectedValue: "10.0.0",
				},
			},
		}
		out := d.Format()
		if !strings.Contains(out, "grafana") {
			t.Errorf("Format() missing subchart name: %s", out)
		}
		if !strings.Contains(out, "9.0.0") {
			t.Errorf("Format() missing actual value: %s", out)
		}
		if !strings.Contains(out, "10.0.0") {
			t.Errorf("Format() missing expected value: %s", out)
		}
	})
}
