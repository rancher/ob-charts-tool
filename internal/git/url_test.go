package git

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsSSHURL(t *testing.T) {
	cases := []struct {
		name string
		url  string
		want bool
	}{
		{"GitHub SSH", "git@github.com:owner/repo.git", true},
		{"GitLab SSH", "git@gitlab.com:owner/repo.git", true},
		{"Generic SSH", "git@example.com:path/to/repo.git", true},
		{"HTTPS URL", "https://github.com/owner/repo.git", false},
		{"HTTP URL", "http://github.com/owner/repo.git", false},
		{"Empty string", "", false},
		{"Random string", "not-a-url", false},
		{"File path", "/path/to/repo", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := IsSSHURL(tc.url)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestIsHTTPSURL(t *testing.T) {
	cases := []struct {
		name string
		url  string
		want bool
	}{
		{"HTTPS GitHub", "https://github.com/owner/repo.git", true},
		{"HTTPS with no .git", "https://github.com/owner/repo", true},
		{"HTTPS GitLab", "https://gitlab.com/owner/repo.git", true},
		{"HTTP URL", "http://github.com/owner/repo.git", false},
		{"SSH URL", "git@github.com:owner/repo.git", false},
		{"Empty string", "", false},
		{"Random string", "not-a-url", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := IsHTTPSURL(tc.url)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestSSHToHTTPS(t *testing.T) {
	cases := []struct {
		name string
		ssh  string
		want string
	}{
		{
			"GitHub SSH with .git",
			"git@github.com:owner/repo.git",
			"https://github.com/owner/repo.git",
		},
		{
			"GitHub SSH without .git",
			"git@github.com:owner/repo",
			"https://github.com/owner/repo.git",
		},
		{
			"Complex path with .git",
			"git@github.com:org/team/project.git",
			"https://github.com/org/team/project.git",
		},
		{
			"Complex path without .git",
			"git@github.com:org/team/project",
			"https://github.com/org/team/project.git",
		},
		{
			"Non-SSH URL returns unchanged",
			"https://github.com/owner/repo.git",
			"https://github.com/owner/repo.git",
		},
		{
			"Random string returns unchanged",
			"not-a-git-url",
			"not-a-git-url",
		},
		{
			"Empty string returns empty",
			"",
			"",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := SSHToHTTPS(tc.ssh)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestNormalizeGitHubURL(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  string
	}{
		{
			"SSH to HTTPS with .git",
			"git@github.com:owner/repo.git",
			"https://github.com/owner/repo.git",
		},
		{
			"SSH to HTTPS without .git adds it",
			"git@github.com:owner/repo",
			"https://github.com/owner/repo.git",
		},
		{
			"HTTPS with .git stays same",
			"https://github.com/owner/repo.git",
			"https://github.com/owner/repo.git",
		},
		{
			"HTTPS without .git adds it",
			"https://github.com/owner/repo",
			"https://github.com/owner/repo.git",
		},
		{
			"Complex nested path SSH",
			"git@github.com:org/team/project.git",
			"https://github.com/org/team/project.git",
		},
		{
			"Complex nested path HTTPS",
			"https://github.com/org/team/project",
			"https://github.com/org/team/project.git",
		},
		{
			"Non-GitHub URL unchanged",
			"https://gitlab.com/owner/repo.git",
			"https://gitlab.com/owner/repo.git",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := NormalizeGitHubURL(tc.input)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestExtractGitHubRepoPath(t *testing.T) {
	cases := []struct {
		name string
		url  string
		want string
	}{
		{
			"GitHub SSH with .git",
			"git@github.com:rancher/ob-team-charts.git",
			"rancher/ob-team-charts",
		},
		{
			"GitHub SSH without .git",
			"git@github.com:rancher/ob-team-charts",
			"rancher/ob-team-charts",
		},
		{
			"GitHub HTTPS with .git",
			"https://github.com/rancher/ob-team-charts.git",
			"rancher/ob-team-charts",
		},
		{
			"GitHub HTTPS without .git",
			"https://github.com/rancher/ob-team-charts",
			"rancher/ob-team-charts",
		},
		{
			"Complex path SSH",
			"git@github.com:org/team/project.git",
			"org/team/project",
		},
		{
			"Complex path HTTPS",
			"https://github.com/org/team/project.git",
			"org/team/project",
		},
		{
			"Non-GitHub SSH returns empty",
			"git@gitlab.com:owner/repo.git",
			"",
		},
		{
			"Non-GitHub HTTPS returns empty",
			"https://gitlab.com/owner/repo.git",
			"",
		},
		{
			"Random string returns empty",
			"not-a-git-url",
			"",
		},
		{
			"Empty string returns empty",
			"",
			"",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ExtractGitHubRepoPath(tc.url)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestIsGitHubRepoURL(t *testing.T) {
	cases := []struct {
		name  string
		url   string
		owner string
		repo  string
		want  bool
	}{
		{
			"SSH URL matches",
			"git@github.com:rancher/ob-team-charts.git",
			"rancher", "ob-team-charts",
			true,
		},
		{
			"HTTPS URL matches",
			"https://github.com/rancher/ob-team-charts.git",
			"rancher", "ob-team-charts",
			true,
		},
		{
			"SSH without .git matches",
			"git@github.com:rancher/ob-team-charts",
			"rancher", "ob-team-charts",
			true,
		},
		{
			"HTTPS without .git matches",
			"https://github.com/rancher/ob-team-charts",
			"rancher", "ob-team-charts",
			true,
		},
		{
			"Case insensitive owner",
			"https://github.com/Rancher/ob-team-charts.git",
			"rancher", "ob-team-charts",
			true,
		},
		{
			"Case insensitive repo",
			"https://github.com/rancher/OB-Team-Charts.git",
			"rancher", "ob-team-charts",
			true,
		},
		{
			"Case insensitive both",
			"https://github.com/RANCHER/OB-TEAM-CHARTS.git",
			"rancher", "ob-team-charts",
			true,
		},
		{
			"Wrong owner",
			"https://github.com/wrong/ob-team-charts.git",
			"rancher", "ob-team-charts",
			false,
		},
		{
			"Wrong repo",
			"https://github.com/rancher/wrong-repo.git",
			"rancher", "ob-team-charts",
			false,
		},
		{
			"Different repo",
			"https://github.com/rancher/helm-charts.git",
			"rancher", "ob-team-charts",
			false,
		},
		{
			"Non-GitHub URL",
			"https://gitlab.com/rancher/ob-team-charts.git",
			"rancher", "ob-team-charts",
			false,
		},
		{
			"Random string",
			"not-a-url",
			"rancher", "ob-team-charts",
			false,
		},
		{
			"Empty URL",
			"",
			"rancher", "ob-team-charts",
			false,
		},
		{
			"Complex nested path",
			"git@github.com:kubernetes/kubernetes.git",
			"kubernetes", "kubernetes",
			true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := IsGitHubRepoURL(tc.url, tc.owner, tc.repo)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestGitHubURLPatterns(t *testing.T) {
	t.Run("GitHubSSHPattern", func(t *testing.T) {
		validSSH := []string{
			"git@github.com:owner/repo",
			"git@github.com:owner/repo.git",
			"git@github.com:org/team/project.git",
		}
		for _, url := range validSSH {
			assert.True(t, GitHubSSHPattern.MatchString(url), "should match: %s", url)
		}

		invalidSSH := []string{
			"https://github.com/owner/repo.git",
			"git@gitlab.com:owner/repo.git",
			"not-a-url",
			"",
		}
		for _, url := range invalidSSH {
			assert.False(t, GitHubSSHPattern.MatchString(url), "should not match: %s", url)
		}
	})

	t.Run("GitHubHTTPSPattern", func(t *testing.T) {
		validHTTPS := []string{
			"https://github.com/owner/repo",
			"https://github.com/owner/repo.git",
			"https://github.com/org/team/project.git",
		}
		for _, url := range validHTTPS {
			assert.True(t, GitHubHTTPSPattern.MatchString(url), "should match: %s", url)
		}

		invalidHTTPS := []string{
			"http://github.com/owner/repo.git",
			"git@github.com:owner/repo.git",
			"https://gitlab.com/owner/repo.git",
			"not-a-url",
			"",
		}
		for _, url := range invalidHTTPS {
			assert.False(t, GitHubHTTPSPattern.MatchString(url), "should not match: %s", url)
		}
	})
}
