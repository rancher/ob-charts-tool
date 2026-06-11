package git_test

import (
	"context"
	"testing"

	"github.com/rancher/ob-charts-tool/helmtools/git"
	"github.com/stretchr/testify/assert"
)

func TestVerifyTagExists_Validation(t *testing.T) {
	ctx := context.Background()

	t.Run("rejects empty repository URL", func(t *testing.T) {
		exists, ref, hash, err := git.VerifyTagExists(ctx, "", "v1.0.0")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot be empty")
		assert.False(t, exists)
		assert.Empty(t, ref)
		assert.Empty(t, hash)
	})

	t.Run("rejects empty tag", func(t *testing.T) {
		exists, ref, hash, err := git.VerifyTagExists(ctx, "https://github.com/test/repo", "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot be empty")
		assert.False(t, exists)
		assert.Empty(t, ref)
		assert.Empty(t, hash)
	})
}

func TestFindMatchingTags_Validation(t *testing.T) {
	ctx := context.Background()

	t.Run("rejects empty repository URL", func(t *testing.T) {
		found, tags, err := git.FindMatchingTags(ctx, "", "v1")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot be empty")
		assert.False(t, found)
		assert.Nil(t, tags)
	})

	t.Run("rejects empty tag pattern", func(t *testing.T) {
		found, tags, err := git.FindMatchingTags(ctx, "https://github.com/test/repo", "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot be empty")
		assert.False(t, found)
		assert.Nil(t, tags)
	})
}

func TestFindHighestVersionTag_Validation(t *testing.T) {
	t.Run("returns nil for empty tags slice", func(t *testing.T) {
		result := git.FindHighestVersionTag([]git.Tag{}, "kube-prometheus-stack")
		assert.Nil(t, result)
	})

	t.Run("returns nil for empty component prefix", func(t *testing.T) {
		tags := []git.Tag{
			{Name: "v1.0.0", Ref: "refs/tags/v1.0.0", CommitHash: "abc123"},
		}
		result := git.FindHighestVersionTag(tags, "")
		assert.Nil(t, result)
	})
}
