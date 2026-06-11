package git_test

import (
	"testing"

	"github.com/rancher/ob-charts-tool/helmtools/git"
)

func TestFindHighestVersionTag_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		tags     []git.Tag
		prefix   string
		wantNil  bool
		wantName string
	}{
		{
			name:    "nil tags slice",
			tags:    nil,
			prefix:  "app",
			wantNil: true,
		},
		{
			name: "tags with mixed valid and invalid versions",
			tags: []git.Tag{
				{Name: "app-1.0.0", Ref: "refs/tags/app-1.0.0", CommitHash: "abc123"},
				{Name: "app-not-a-version", Ref: "refs/tags/app-not-a-version", CommitHash: "def456"},
				{Name: "app-2.0.0-beta+build", Ref: "refs/tags/app-2.0.0-beta+build", CommitHash: "ghi789"},
				{Name: "app-1.5.0", Ref: "refs/tags/app-1.5.0", CommitHash: "jkl012"},
			},
			prefix:   "app",
			wantNil:  false,
			wantName: "app-2.0.0-beta+build",
		},
		{
			name: "only invalid versions",
			tags: []git.Tag{
				{Name: "app-invalid1", Ref: "refs/tags/app-invalid1", CommitHash: "abc"},
				{Name: "app-invalid2", Ref: "refs/tags/app-invalid2", CommitHash: "def"},
				{Name: "app-not-semver", Ref: "refs/tags/app-not-semver", CommitHash: "ghi"},
			},
			prefix:  "app",
			wantNil: true,
		},
		{
			name: "tags without prefix",
			tags: []git.Tag{
				{Name: "v1.0.0", Ref: "refs/tags/v1.0.0", CommitHash: "abc"},
				{Name: "v2.0.0", Ref: "refs/tags/v2.0.0", CommitHash: "def"},
			},
			prefix:  "app",
			wantNil: true,
		},
		{
			name: "mixed prefixes",
			tags: []git.Tag{
				{Name: "app-1.0.0", Ref: "refs/tags/app-1.0.0", CommitHash: "abc"},
				{Name: "other-5.0.0", Ref: "refs/tags/other-5.0.0", CommitHash: "def"},
				{Name: "app-2.0.0", Ref: "refs/tags/app-2.0.0", CommitHash: "ghi"},
			},
			prefix:   "app",
			wantNil:  false,
			wantName: "app-2.0.0",
		},
		{
			name: "versions with leading zeros",
			tags: []git.Tag{
				{Name: "app-1.0.0", Ref: "refs/tags/app-1.0.0", CommitHash: "abc"},
				{Name: "app-1.01.0", Ref: "refs/tags/app-1.01.0", CommitHash: "def"},
			},
			prefix:  "app",
			wantNil: false,
			// semver should handle 1.01.0 as invalid or normalize it
		},
		{
			name: "very long version numbers",
			tags: []git.Tag{
				{Name: "app-999.999.999", Ref: "refs/tags/app-999.999.999", CommitHash: "abc"},
				{Name: "app-1000.0.0", Ref: "refs/tags/app-1000.0.0", CommitHash: "def"},
			},
			prefix:   "app",
			wantNil:  false,
			wantName: "app-1000.0.0",
		},
		{
			name: "prerelease comparison",
			tags: []git.Tag{
				{Name: "app-1.0.0-alpha", Ref: "refs/tags/app-1.0.0-alpha", CommitHash: "abc"},
				{Name: "app-1.0.0-beta", Ref: "refs/tags/app-1.0.0-beta", CommitHash: "def"},
				{Name: "app-1.0.0-rc.1", Ref: "refs/tags/app-1.0.0-rc.1", CommitHash: "ghi"},
			},
			prefix:  "app",
			wantNil: false,
			// rc.1 > beta > alpha in semver
			wantName: "app-1.0.0-rc.1",
		},
		{
			name: "release vs prerelease",
			tags: []git.Tag{
				{Name: "app-1.0.0", Ref: "refs/tags/app-1.0.0", CommitHash: "abc"},
				{Name: "app-1.0.1-alpha", Ref: "refs/tags/app-1.0.1-alpha", CommitHash: "def"},
			},
			prefix:  "app",
			wantNil: false,
			// 1.0.1-alpha > 1.0.0 in semver
			wantName: "app-1.0.1-alpha",
		},
		{
			name: "tags with same version different build metadata",
			tags: []git.Tag{
				{Name: "app-1.0.0+build1", Ref: "refs/tags/app-1.0.0+build1", CommitHash: "abc"},
				{Name: "app-1.0.0+build2", Ref: "refs/tags/app-1.0.0+build2", CommitHash: "def"},
			},
			prefix:  "app",
			wantNil: false,
			// Build metadata doesn't affect precedence, so either could be returned
		},
		{
			name: "single tag with exact prefix match",
			tags: []git.Tag{
				{Name: "app-1.0.0", Ref: "refs/tags/app-1.0.0", CommitHash: "abc"},
			},
			prefix:   "app",
			wantNil:  false,
			wantName: "app-1.0.0",
		},
		{
			name: "tag name exactly equals prefix (no version)",
			tags: []git.Tag{
				{Name: "app", Ref: "refs/tags/app", CommitHash: "abc"},
				{Name: "app-1.0.0", Ref: "refs/tags/app-1.0.0", CommitHash: "def"},
			},
			prefix:   "app",
			wantNil:  false,
			wantName: "app-1.0.0",
		},
		{
			name: "whitespace in tag names",
			tags: []git.Tag{
				{Name: "app-1.0.0 ", Ref: "refs/tags/app-1.0.0", CommitHash: "abc"},
				{Name: " app-2.0.0", Ref: "refs/tags/app-2.0.0", CommitHash: "def"},
			},
			prefix:  "app",
			wantNil: true, // Neither should match due to whitespace
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := git.FindHighestVersionTag(tt.tags, tt.prefix)
			if tt.wantNil {
				if result != nil {
					t.Errorf("FindHighestVersionTag() = %+v, want nil", result)
				}
				return
			}
			if result == nil {
				t.Fatal("FindHighestVersionTag() = nil, want non-nil")
			}
			if tt.wantName != "" && result.Name != tt.wantName {
				t.Errorf("FindHighestVersionTag().Name = %v, want %v", result.Name, tt.wantName)
			}
		})
	}
}

func TestFindHighestVersionTag_Stability(t *testing.T) {
	// Test that the function is deterministic when called multiple times
	tags := []git.Tag{
		{Name: "app-1.0.0", Ref: "refs/tags/app-1.0.0", CommitHash: "abc"},
		{Name: "app-2.0.0", Ref: "refs/tags/app-2.0.0", CommitHash: "def"},
		{Name: "app-1.5.0", Ref: "refs/tags/app-1.5.0", CommitHash: "ghi"},
	}

	var firstResult *git.Tag
	for i := 0; i < 10; i++ {
		result := git.FindHighestVersionTag(tags, "app")
		if result == nil {
			t.Fatal("FindHighestVersionTag() returned nil")
		}
		if firstResult == nil {
			firstResult = result
		} else {
			if result.Name != firstResult.Name {
				t.Errorf("FindHighestVersionTag() inconsistent: iteration %d returned %s, first returned %s",
					i, result.Name, firstResult.Name)
			}
		}
	}

	if firstResult.Name != "app-2.0.0" {
		t.Errorf("FindHighestVersionTag() = %s, want app-2.0.0", firstResult.Name)
	}
}

func BenchmarkFindHighestVersionTag(b *testing.B) {
	// Test with a large number of tags
	interactions := b.N
	b.Logf("b.N = %d", interactions)
	tags := make([]git.Tag, interactions)
	for i := range interactions {
		// Create tags with various patterns
		var name string
		if i%10 == 0 {
			// Some invalid tags
			name = "app-invalid"
		} else {
			// Valid semver tags
			major := i / 100
			minor := (i % 100) / 10
			patch := i % 10
			name = "app-" + string(rune('0'+major)) + "." + string(rune('0'+minor)) + "." + string(rune('0'+patch))
		}
		tags[i] = git.Tag{
			Name:       name,
			Ref:        "refs/tags/" + name,
			CommitHash: "hash" + string(rune('0'+i%10)),
		}
	}

	_ = git.FindHighestVersionTag(tags, "app")
}

func TestTag_Structure(t *testing.T) {
	// Test that Tag struct can be created and accessed
	tag := git.Tag{
		Name:       "test-1.0.0",
		Ref:        "refs/tags/test-1.0.0",
		CommitHash: "abc123def456",
	}

	if tag.Name != "test-1.0.0" {
		t.Errorf("Tag.Name = %s, want test-1.0.0", tag.Name)
	}
	if tag.Ref != "refs/tags/test-1.0.0" {
		t.Errorf("Tag.Ref = %s, want refs/tags/test-1.0.0", tag.Ref)
	}
	if tag.CommitHash != "abc123def456" {
		t.Errorf("Tag.CommitHash = %s, want abc123def456", tag.CommitHash)
	}
}

func TestTag_ZeroValue(t *testing.T) {
	// Test that zero value of Tag is safe to use
	var tag git.Tag

	if tag.Name != "" {
		t.Errorf("Zero Tag.Name = %s, want empty string", tag.Name)
	}
	if tag.Ref != "" {
		t.Errorf("Zero Tag.Ref = %s, want empty string", tag.Ref)
	}
	if tag.CommitHash != "" {
		t.Errorf("Zero Tag.CommitHash = %s, want empty string", tag.CommitHash)
	}

	// Should be safe to use in FindHighestVersionTag
	tags := []git.Tag{tag}
	result := git.FindHighestVersionTag(tags, "prefix")
	if result != nil {
		t.Errorf("FindHighestVersionTag() with zero-value tag = %+v, want nil", result)
	}
}
