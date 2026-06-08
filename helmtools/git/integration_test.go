//go:build integration
// +build integration

package git_test

import (
	"context"
	"testing"

	"github.com/rancher/ob-charts-tool/helmtools/git"
)

// Integration tests make real network calls to public Git repositories.
// They are excluded from the default test suite and must be run explicitly.
//
// Run with:
//   go test -tags=integration ./helmtools/git/
//   make test-integration
//   ./scripts/test-integration
//
// These tests are skipped in CI by default but can be enabled for testing
// network operations and validating behavior against real repositories.

const (
	testRepo = "https://github.com/prometheus-community/helm-charts"
	testTag  = "kube-prometheus-stack-65.8.1"
)

func TestVerifyTagExists_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tests := []struct {
		name       string
		repoURL    string
		tag        string
		wantExists bool
		wantErr    bool
	}{
		{
			name:       "existing tag",
			repoURL:    testRepo,
			tag:        testTag,
			wantExists: true,
			wantErr:    false,
		},
		{
			name:       "non-existent tag",
			repoURL:    testRepo,
			tag:        "non-existent-tag-xyz-123",
			wantExists: false,
			wantErr:    false,
		},
		{
			name:       "invalid repository",
			repoURL:    "https://github.com/nonexistent/repo-xyz-123",
			wantExists: false,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exists, ref, hash, err := git.VerifyTagExists(context.Background(), tt.repoURL, tt.tag)
			if (err != nil) != tt.wantErr {
				t.Errorf("VerifyTagExists() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if exists != tt.wantExists {
				t.Errorf("VerifyTagExists() exists = %v, want %v", exists, tt.wantExists)
			}
			if tt.wantExists {
				if ref == "" {
					t.Error("VerifyTagExists() ref should not be empty for existing tag")
				}
				if hash == "" {
					t.Error("VerifyTagExists() hash should not be empty for existing tag")
				}
			}
		})
	}
}

func TestFindMatchingTags_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tests := []struct {
		name        string
		repoURL     string
		tagPartial  string
		wantFound   bool
		wantMinTags int
		wantErr     bool
	}{
		{
			name:        "find kube-prometheus-stack tags",
			repoURL:     testRepo,
			tagPartial:  "kube-prometheus-stack-",
			wantFound:   true,
			wantMinTags: 1,
			wantErr:     false,
		},
		{
			name:        "no matching tags",
			repoURL:     testRepo,
			tagPartial:  "nonexistent-chart-xyz-",
			wantFound:   false,
			wantMinTags: 0,
			wantErr:     false,
		},
		{
			name:        "invalid repository",
			repoURL:     "https://github.com/nonexistent/repo-xyz-123",
			tagPartial:  "any",
			wantFound:   false,
			wantMinTags: 0,
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			found, tags, err := git.FindMatchingTags(context.Background(), tt.repoURL, tt.tagPartial)
			if (err != nil) != tt.wantErr {
				t.Errorf("FindMatchingTags() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if found != tt.wantFound {
				t.Errorf("FindMatchingTags() found = %v, want %v", found, tt.wantFound)
			}
			if len(tags) < tt.wantMinTags {
				t.Errorf("FindMatchingTags() returned %d tags, want at least %d", len(tags), tt.wantMinTags)
			}
			if tt.wantFound {
				// Verify tag structure
				for i, tag := range tags {
					if tag.Name == "" {
						t.Errorf("FindMatchingTags() tag[%d].Name is empty", i)
					}
					if tag.Ref == "" {
						t.Errorf("FindMatchingTags() tag[%d].Ref is empty", i)
					}
					if tag.CommitHash == "" {
						t.Errorf("FindMatchingTags() tag[%d].CommitHash is empty", i)
					}
				}
			}
		})
	}
}

func TestFindHighestVersionTag_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	t.Run("find highest from real repository", func(t *testing.T) {
		found, tags, err := git.FindMatchingTags(context.Background(), testRepo, "kube-prometheus-stack-")
		if err != nil {
			t.Fatalf("FindMatchingTags() error = %v", err)
		}
		if !found {
			t.Fatal("FindMatchingTags() found no tags")
		}

		highest := git.FindHighestVersionTag(tags, "kube-prometheus-stack")
		if highest == nil {
			t.Fatal("FindHighestVersionTag() returned nil")
		}

		if highest.Name == "" {
			t.Error("FindHighestVersionTag() returned tag with empty name")
		}
		if highest.CommitHash == "" {
			t.Error("FindHighestVersionTag() returned tag with empty hash")
		}

		t.Logf("Highest version tag: %s (hash: %s)", highest.Name, highest.CommitHash)
	})
}
