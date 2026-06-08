package git

// Tag represents a git tag with its associated metadata.
type Tag struct {
	Name       string // Short tag name (e.g., "v1.0.0")
	Ref        string // Full reference (e.g., "refs/tags/v1.0.0")
	CommitHash string // Commit hash the tag points to
}
