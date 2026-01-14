package git

import (
	"regexp"
	"strings"
)

// GitHubSSHPattern matches SSH-style GitHub URLs like git@github.com:owner/repo.git
var GitHubSSHPattern = regexp.MustCompile(`^git@github\.com:(.+?)(?:\.git)?$`)

// GitHubHTTPSPattern matches HTTPS GitHub URLs like https://github.com/owner/repo.git
var GitHubHTTPSPattern = regexp.MustCompile(`^https://github\.com/(.+?)(?:\.git)?$`)

// IsSSHURL returns true if the URL is an SSH-style git URL
func IsSSHURL(url string) bool {
	return strings.HasPrefix(url, "git@")
}

// IsHTTPSURL returns true if the URL is an HTTPS git URL
func IsHTTPSURL(url string) bool {
	return strings.HasPrefix(url, "https://")
}

// SSHToHTTPS converts an SSH git URL to HTTPS format.
// For example: git@github.com:owner/repo.git -> https://github.com/owner/repo.git
// If the URL is not an SSH URL or can't be converted, returns the original URL.
func SSHToHTTPS(url string) string {
	matches := GitHubSSHPattern.FindStringSubmatch(url)
	if len(matches) < 2 {
		return url
	}

	path := matches[1]
	// Ensure .git suffix for consistency
	if !strings.HasSuffix(path, ".git") {
		path = path + ".git"
	}

	return "https://github.com/" + path
}

// NormalizeGitHubURL normalizes a GitHub URL to a canonical HTTPS format.
// This handles both SSH and HTTPS URLs, with or without .git suffix.
// Returns the normalized URL in the format: https://github.com/owner/repo.git
func NormalizeGitHubURL(url string) string {
	// First convert SSH to HTTPS if needed
	if IsSSHURL(url) {
		url = SSHToHTTPS(url)
	}

	// Ensure .git suffix for consistency
	if IsHTTPSURL(url) && !strings.HasSuffix(url, ".git") {
		url = url + ".git"
	}

	return url
}

// ExtractGitHubRepoPath extracts the owner/repo path from a GitHub URL.
// Works with both SSH and HTTPS URLs.
// Returns empty string if the URL is not a valid GitHub URL.
func ExtractGitHubRepoPath(url string) string {
	// Try SSH pattern first
	if matches := GitHubSSHPattern.FindStringSubmatch(url); len(matches) >= 2 {
		path := matches[1]
		return strings.TrimSuffix(path, ".git")
	}

	// Try HTTPS pattern
	if matches := GitHubHTTPSPattern.FindStringSubmatch(url); len(matches) >= 2 {
		path := matches[1]
		return strings.TrimSuffix(path, ".git")
	}

	return ""
}

// IsGitHubRepoURL checks if the URL points to a specific GitHub repository.
// Compares URLs in a normalized way, handling both SSH and HTTPS formats.
func IsGitHubRepoURL(url string, owner string, repo string) bool {
	repoPath := ExtractGitHubRepoPath(url)
	if repoPath == "" {
		return false
	}

	expected := strings.ToLower(owner + "/" + repo)
	return strings.ToLower(repoPath) == expected
}
