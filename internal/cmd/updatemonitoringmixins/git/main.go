package git

import "github.com/go-git/go-git/v5/plumbing"

type RepoConfigStatus struct {
	Name    string `json:"name"`
	RepoURL string `json:"repo_url"`
	HeadSha string `json:"head_sha"`
}

func (repoConfig *RepoConfigStatus) HeadHash() plumbing.Hash {
	return plumbing.NewHash(repoConfig.HeadSha)
}

func RepoSHAs(repos []RepoConfigStatus) map[string]string {
	refs := make(map[string]string, len(repos))
	for _, repo := range repos {
		refs[repo.Name] = repo.HeadSha
	}

	return refs
}
