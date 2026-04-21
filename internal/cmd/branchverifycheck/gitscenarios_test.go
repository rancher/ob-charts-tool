package branchverifycheck

// gitscenarios_test.go contains tests for the git-refs-dependent check functions
// (FindModifiedPackages, CheckBranchCurrent, CountCommitsBehind) using realistic
// mock git repositories built entirely on disk with no network calls.
//
// The core concept: we use go-git's PlainInit to create a real git repo on disk,
// create commits with specific file contents, then manually construct GitRefs from
// those commit objects — bypassing EnsureUpstreamRemote (which needs GitHub).

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// =============================================================================
// Mock repo builder
// =============================================================================

// mockRepo wraps a go-git repository backed by a temp-dir on disk.
// It provides helpers to write files, create commits, and build GitRefs
// for use in tests — all without any network activity.
type mockRepo struct {
	dir  string
	repo *git.Repository
	wt   *git.Worktree
	sig  *object.Signature
}

// newMockRepo initialises a fresh git repo in a temp directory.
func newMockRepo(t *testing.T) *mockRepo {
	t.Helper()
	dir := t.TempDir()
	repo, err := git.PlainInit(dir, false)
	if err != nil {
		t.Fatalf("PlainInit: %v", err)
	}
	wt, err := repo.Worktree()
	if err != nil {
		t.Fatalf("Worktree: %v", err)
	}
	return &mockRepo{
		dir:  dir,
		repo: repo,
		wt:   wt,
		sig:  &object.Signature{Name: "Test", Email: "test@test.com", When: time.Now()},
	}
}

// commit writes the given files to disk, stages them all, and creates a commit.
// files is a map of repo-relative path → content.
// Returns the resulting *object.Commit.
func (m *mockRepo) commit(t *testing.T, files map[string]string, msg string) *object.Commit {
	t.Helper()
	for relPath, content := range files {
		writeFile(t, fmt.Sprintf("%s/%s", m.dir, relPath), content)
		if _, err := m.wt.Add(relPath); err != nil {
			t.Fatalf("git add %s: %v", relPath, err)
		}
	}
	hash, err := m.wt.Commit(msg, &git.CommitOptions{Author: m.sig})
	if err != nil {
		t.Fatalf("git commit %q: %v", msg, err)
	}
	c, err := m.repo.CommitObject(hash)
	if err != nil {
		t.Fatalf("CommitObject: %v", err)
	}
	return c
}

// makeRef creates a named reference (branch-style) pointing to the given commit hash.
// This lets us build synthetic UpstreamRef / HeadRef values without a real remote.
func (m *mockRepo) makeRef(t *testing.T, name string, c *object.Commit) *plumbing.Reference {
	t.Helper()
	ref := plumbing.NewHashReference(plumbing.ReferenceName(name), c.Hash)
	if err := m.repo.Storer.SetReference(ref); err != nil {
		t.Fatalf("SetReference %s: %v", name, err)
	}
	stored, err := m.repo.Reference(plumbing.ReferenceName(name), true)
	if err != nil {
		t.Fatalf("Reference %s: %v", name, err)
	}
	return stored
}

// buildRefs manually assembles a GitRefs from real commit objects.
// upstreamRefName is the name to use for the synthetic upstream reference
// (e.g. "refs/remotes/upstream/main").
func (m *mockRepo) buildRefs(
	t *testing.T,
	head, upstream, mergeBase *object.Commit,
	headRefName, upstreamRefName string,
) *GitRefs {
	t.Helper()
	headRef := m.makeRef(t, headRefName, head)
	upstreamRef := m.makeRef(t, upstreamRefName, upstream)
	return &GitRefs{
		HeadRef:         headRef,
		HeadCommit:      head,
		UpstreamRef:     upstreamRef,
		UpstreamCommit:  upstream,
		MergeBaseCommit: mergeBase,
	}
}

// =============================================================================
// TestFindModifiedPackages
// =============================================================================

// TestFindModifiedPackages verifies that FindModifiedPackages correctly identifies
// which packages (under packages/<name>/<version>/) were modified between the
// branch's merge-base and HEAD.
func TestFindModifiedPackages(t *testing.T) {
	cases := []struct {
		name string
		// baseFiles are committed first (this becomes the merge-base).
		baseFiles map[string]string
		// branchFiles are committed on top (this becomes HEAD).
		branchFiles  map[string]string
		wantCount    int
		wantPassed   bool     // CheckResult.Passed
		wantPackages []string // FullPath values expected (order-independent check)
	}{
		{
			name: "single package modified",
			baseFiles: map[string]string{
				"README.md": "# repo\n",
				"packages/rancher-logging/4.1/package.yaml": "version: 1.0.0-rancher.1\n",
			},
			branchFiles: map[string]string{
				"packages/rancher-logging/4.1/package.yaml": "version: 1.0.0-rancher.2\n",
			},
			wantCount:    1,
			wantPassed:   true,
			wantPackages: []string{"rancher-logging/4.1"},
		},
		{
			name: "no packages modified (only README changed)",
			baseFiles: map[string]string{
				"README.md": "# repo\n",
			},
			branchFiles: map[string]string{
				"README.md": "# updated\n",
			},
			wantCount:  0,
			wantPassed: true,
		},
		{
			name: "multiple packages modified (warning)",
			baseFiles: map[string]string{
				"packages/rancher-logging/4.1/package.yaml":     "version: 1.0.0-rancher.1\n",
				"packages/rancher-monitoring/77.9/package.yaml": "version: 77.0.0-rancher.1\n",
			},
			branchFiles: map[string]string{
				"packages/rancher-logging/4.1/package.yaml":     "version: 1.0.0-rancher.2\n",
				"packages/rancher-monitoring/77.9/package.yaml": "version: 77.0.0-rancher.2\n",
			},
			wantCount:    2,
			wantPassed:   false, // multiple packages → soft fail / warning
			wantPackages: []string{"rancher-logging/4.1", "rancher-monitoring/77.9"},
		},
		{
			name: "new file added in package directory",
			baseFiles: map[string]string{
				"README.md": "# repo\n",
			},
			branchFiles: map[string]string{
				"packages/rancher-project-monitoring/0.3/package.yaml": "version: 0.3.0\n",
			},
			wantCount:    1,
			wantPassed:   true,
			wantPackages: []string{"rancher-project-monitoring/0.3"},
		},
		{
			name: "changes outside packages/ dir not counted",
			baseFiles: map[string]string{
				"README.md": "# repo\n",
			},
			branchFiles: map[string]string{
				"README.md":        "# updated\n",
				".github/ci.yml":   "name: ci\n",
				"charts/README.md": "# charts\n",
			},
			wantCount:  0,
			wantPassed: true,
		},
		{
			name: "multiple file changes within same package: counted once",
			baseFiles: map[string]string{
				"packages/rancher-logging/4.1/package.yaml": "version: 1.0.0\n",
			},
			branchFiles: map[string]string{
				"packages/rancher-logging/4.1/package.yaml": "version: 1.0.0-rancher.2\n",
				"packages/rancher-logging/4.1/patch.yaml":   "# patch\n",
			},
			wantCount:    1,
			wantPassed:   true,
			wantPackages: []string{"rancher-logging/4.1"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := newMockRepo(t)

			// Create the base commit (simulates upstream/main state = merge-base)
			mergeBase := m.commit(t, tc.baseFiles, "base commit")

			// Create the branch commit (simulates HEAD on the feature branch)
			head := m.commit(t, tc.branchFiles, "branch changes")

			refs := m.buildRefs(t, head, mergeBase, mergeBase,
				"refs/heads/my-feature", "refs/remotes/upstream/main")

			packages, result := FindModifiedPackages(refs)

			if result.Passed != tc.wantPassed {
				t.Errorf("FindModifiedPackages.Passed = %v, want %v: %s", result.Passed, tc.wantPassed, result.Message)
			}

			if len(packages) != tc.wantCount {
				t.Errorf("len(packages) = %d, want %d: %v", len(packages), tc.wantCount, packages)
				return
			}

			if len(tc.wantPackages) > 0 {
				found := make(map[string]bool)
				for _, p := range packages {
					found[p.FullPath] = true
				}
				for _, want := range tc.wantPackages {
					if !found[want] {
						t.Errorf("expected package %q not found in results: %v", want, packages)
					}
				}
			}
		})
	}
}

// =============================================================================
// TestCheckBranchCurrent
// =============================================================================

// TestCheckBranchCurrent verifies that CheckBranchCurrent correctly reports
// whether the branch is up-to-date with upstream/main, and counts how many
// commits behind it is.
func TestCheckBranchCurrent(t *testing.T) {
	t.Run("up-to-date: merge-base equals upstream", func(t *testing.T) {
		m := newMockRepo(t)

		// Single shared commit: merge-base and upstream are the same.
		base := m.commit(t, map[string]string{"README.md": "# repo\n"}, "shared base")

		refs := m.buildRefs(t, base, base, base,
			"refs/heads/my-feature", "refs/remotes/upstream/main")

		info, result := CheckBranchCurrent(refs, m.repo)

		if !result.Passed {
			t.Errorf("CheckBranchCurrent should pass when branch is up-to-date: %s", result.Message)
		}
		if !info.IsUpToDate {
			t.Errorf("BranchInfo.IsUpToDate should be true")
		}
		if info.CommitsBehind != 0 {
			t.Errorf("CommitsBehind = %d, want 0", info.CommitsBehind)
		}
	})

	t.Run("behind upstream by 1 commit", func(t *testing.T) {
		m := newMockRepo(t)

		mergeBase := m.commit(t, map[string]string{"README.md": "# repo\n"}, "shared base")

		// One upstream commit that the branch doesn't have
		upstream := m.commit(t, map[string]string{"README.md": "# upstream update\n"}, "upstream commit")

		// Simulate the branch still sitting at mergeBase (HEAD = mergeBase)
		refs := m.buildRefs(t, mergeBase, upstream, mergeBase,
			"refs/heads/my-feature", "refs/remotes/upstream/main")

		info, result := CheckBranchCurrent(refs, m.repo)

		if result.Passed {
			t.Errorf("CheckBranchCurrent should fail (warn) when branch is behind")
		}
		if info.IsUpToDate {
			t.Errorf("BranchInfo.IsUpToDate should be false")
		}
		if info.CommitsBehind != 1 {
			t.Errorf("CommitsBehind = %d, want 1", info.CommitsBehind)
		}
	})

	t.Run("behind upstream by 3 commits", func(t *testing.T) {
		m := newMockRepo(t)

		mergeBase := m.commit(t, map[string]string{"README.md": "# base\n"}, "shared base")
		m.commit(t, map[string]string{"a.txt": "1\n"}, "upstream +1")
		m.commit(t, map[string]string{"b.txt": "2\n"}, "upstream +2")
		upstream := m.commit(t, map[string]string{"c.txt": "3\n"}, "upstream +3")

		refs := m.buildRefs(t, mergeBase, upstream, mergeBase,
			"refs/heads/my-feature", "refs/remotes/upstream/main")

		info, result := CheckBranchCurrent(refs, m.repo)

		if result.Passed {
			t.Errorf("CheckBranchCurrent should fail when branch is behind")
		}
		if info.CommitsBehind != 3 {
			t.Errorf("CommitsBehind = %d, want 3", info.CommitsBehind)
		}
		if !strings.Contains(result.Message, "3") {
			t.Errorf("message should mention commit count: %s", result.Message)
		}
	})
}

// =============================================================================
// TestCountCommitsBehind
// =============================================================================

func TestCountCommitsBehind(t *testing.T) {
	cases := []struct {
		name          string
		upstreamExtra int // how many extra commits to add after mergeBase
		want          int
	}{
		{"0 commits behind (up-to-date)", 0, 0},
		{"1 commit behind", 1, 1},
		{"5 commits behind", 5, 5},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := newMockRepo(t)

			mergeBase := m.commit(t, map[string]string{"README.md": "# base\n"}, "merge-base")

			var upstream *object.Commit
			upstream = mergeBase
			for i := range tc.upstreamExtra {
				upstream = m.commit(t, map[string]string{
					fmt.Sprintf("file%d.txt", i): "content\n",
				}, fmt.Sprintf("upstream commit %d", i+1))
			}

			upstreamRef := m.makeRef(t, "refs/remotes/upstream/main", upstream)

			got := CountCommitsBehind(m.repo, upstreamRef, mergeBase.Hash)
			if got != tc.want {
				t.Errorf("CountCommitsBehind = %d, want %d", got, tc.want)
			}
		})
	}
}
