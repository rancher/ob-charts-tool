package git_test

import (
	"testing"

	"github.com/rancher/ob-charts-tool/helmtools/git"
)

func TestFindHighestVersionTag(t *testing.T) {
	tests := []struct {
		name            string
		tags            []git.Tag
		componentPrefix string
		wantTag         *git.Tag
	}{
		{
			name: "finds highest semantic version",
			tags: []git.Tag{
				{Name: "kube-prometheus-stack-1.0.0", Ref: "refs/tags/kube-prometheus-stack-1.0.0", CommitHash: "hash1"},
				{Name: "kube-prometheus-stack-2.0.0", Ref: "refs/tags/kube-prometheus-stack-2.0.0", CommitHash: "hash2"},
				{Name: "kube-prometheus-stack-1.5.0", Ref: "refs/tags/kube-prometheus-stack-1.5.0", CommitHash: "hash3"},
			},
			componentPrefix: "kube-prometheus-stack",
			wantTag:         &git.Tag{Name: "kube-prometheus-stack-2.0.0", Ref: "refs/tags/kube-prometheus-stack-2.0.0", CommitHash: "hash2"},
		},
		{
			name: "handles patch versions correctly",
			tags: []git.Tag{
				{Name: "app-1.2.3", Ref: "refs/tags/app-1.2.3", CommitHash: "hash1"},
				{Name: "app-1.2.10", Ref: "refs/tags/app-1.2.10", CommitHash: "hash2"},
				{Name: "app-1.2.5", Ref: "refs/tags/app-1.2.5", CommitHash: "hash3"},
			},
			componentPrefix: "app",
			wantTag:         &git.Tag{Name: "app-1.2.10", Ref: "refs/tags/app-1.2.10", CommitHash: "hash2"},
		},
		{
			name: "ignores tags with different prefix",
			tags: []git.Tag{
				{Name: "other-component-5.0.0", Ref: "refs/tags/other-component-5.0.0", CommitHash: "hash1"},
				{Name: "my-component-1.0.0", Ref: "refs/tags/my-component-1.0.0", CommitHash: "hash2"},
				{Name: "my-component-2.0.0", Ref: "refs/tags/my-component-2.0.0", CommitHash: "hash3"},
			},
			componentPrefix: "my-component",
			wantTag:         &git.Tag{Name: "my-component-2.0.0", Ref: "refs/tags/my-component-2.0.0", CommitHash: "hash3"},
		},
		{
			name: "ignores invalid semantic versions",
			tags: []git.Tag{
				{Name: "app-1.0.0", Ref: "refs/tags/app-1.0.0", CommitHash: "hash1"},
				{Name: "app-invalid", Ref: "refs/tags/app-invalid", CommitHash: "hash2"},
				{Name: "app-2.0.0", Ref: "refs/tags/app-2.0.0", CommitHash: "hash3"},
			},
			componentPrefix: "app",
			wantTag:         &git.Tag{Name: "app-2.0.0", Ref: "refs/tags/app-2.0.0", CommitHash: "hash3"},
		},
		{
			name: "returns nil when no matching tags",
			tags: []git.Tag{
				{Name: "other-1.0.0", Ref: "refs/tags/other-1.0.0", CommitHash: "hash1"},
				{Name: "different-2.0.0", Ref: "refs/tags/different-2.0.0", CommitHash: "hash2"},
			},
			componentPrefix: "my-component",
			wantTag:         nil,
		},
		{
			name: "returns nil when all versions are invalid",
			tags: []git.Tag{
				{Name: "app-invalid1", Ref: "refs/tags/app-invalid1", CommitHash: "hash1"},
				{Name: "app-invalid2", Ref: "refs/tags/app-invalid2", CommitHash: "hash2"},
			},
			componentPrefix: "app",
			wantTag:         nil,
		},
		{
			name:            "returns nil for empty tags slice",
			tags:            []git.Tag{},
			componentPrefix: "app",
			wantTag:         nil,
		},
		{
			name: "returns nil for empty component prefix",
			tags: []git.Tag{
				{Name: "app-1.0.0", Ref: "refs/tags/app-1.0.0", CommitHash: "hash1"},
			},
			componentPrefix: "",
			wantTag:         nil,
		},
		{
			name: "handles prerelease versions",
			tags: []git.Tag{
				{Name: "app-1.0.0-alpha", Ref: "refs/tags/app-1.0.0-alpha", CommitHash: "hash1"},
				{Name: "app-1.0.0-beta", Ref: "refs/tags/app-1.0.0-beta", CommitHash: "hash2"},
				{Name: "app-1.0.0", Ref: "refs/tags/app-1.0.0", CommitHash: "hash3"},
			},
			componentPrefix: "app",
			wantTag:         &git.Tag{Name: "app-1.0.0", Ref: "refs/tags/app-1.0.0", CommitHash: "hash3"},
		},
		{
			name: "compares major versions correctly",
			tags: []git.Tag{
				{Name: "app-10.0.0", Ref: "refs/tags/app-10.0.0", CommitHash: "hash1"},
				{Name: "app-2.0.0", Ref: "refs/tags/app-2.0.0", CommitHash: "hash2"},
				{Name: "app-9.0.0", Ref: "refs/tags/app-9.0.0", CommitHash: "hash3"},
			},
			componentPrefix: "app",
			wantTag:         &git.Tag{Name: "app-10.0.0", Ref: "refs/tags/app-10.0.0", CommitHash: "hash1"},
		},
		{
			name: "handles v prefix in versions",
			tags: []git.Tag{
				{Name: "app-v1.0.0", Ref: "refs/tags/app-v1.0.0", CommitHash: "hash1"},
				{Name: "app-v2.0.0", Ref: "refs/tags/app-v2.0.0", CommitHash: "hash2"},
				{Name: "app-v1.5.0", Ref: "refs/tags/app-v1.5.0", CommitHash: "hash3"},
			},
			componentPrefix: "app",
			wantTag:         &git.Tag{Name: "app-v2.0.0", Ref: "refs/tags/app-v2.0.0", CommitHash: "hash2"},
		},
		{
			name: "single valid tag",
			tags: []git.Tag{
				{Name: "app-1.0.0", Ref: "refs/tags/app-1.0.0", CommitHash: "hash1"},
			},
			componentPrefix: "app",
			wantTag:         &git.Tag{Name: "app-1.0.0", Ref: "refs/tags/app-1.0.0", CommitHash: "hash1"},
		},
		{
			name: "handles build metadata",
			tags: []git.Tag{
				{Name: "app-1.0.0+build1", Ref: "refs/tags/app-1.0.0+build1", CommitHash: "hash1"},
				{Name: "app-1.0.0+build2", Ref: "refs/tags/app-1.0.0+build2", CommitHash: "hash2"},
				{Name: "app-2.0.0", Ref: "refs/tags/app-2.0.0", CommitHash: "hash3"},
			},
			componentPrefix: "app",
			wantTag:         &git.Tag{Name: "app-2.0.0", Ref: "refs/tags/app-2.0.0", CommitHash: "hash3"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := git.FindHighestVersionTag(tt.tags, tt.componentPrefix)
			if tt.wantTag == nil {
				if got != nil {
					t.Errorf("FindHighestVersionTag() = %+v, want nil", got)
				}
				return
			}
			if got == nil {
				t.Errorf("FindHighestVersionTag() = nil, want %+v", tt.wantTag)
				return
			}
			if got.Name != tt.wantTag.Name || got.Ref != tt.wantTag.Ref || got.CommitHash != tt.wantTag.CommitHash {
				t.Errorf("FindHighestVersionTag() = %+v, want %+v", got, tt.wantTag)
			}
		})
	}
}
