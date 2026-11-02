package github

import (
	"fmt"

	"github.com/google/go-github/github"
)

// GetReleaseURL get release url
func (c *Client) GetReleaseURL(owner, repo, tag string) string {
	return fmt.Sprintf("https://github.com/%s/%s/releases/tag/%s", owner, repo, tag)
}

// GetReleaseByTag get release
func (c *Client) GetReleaseByTag(owner, repo, tag string) (*github.RepositoryRelease, error) {
	resp, _, err := c.svc.Repositories.GetReleaseByTag(c.ctx, owner, repo, tag)
	if err != nil {
		return nil, err
	}
	return resp, err
}

// GetTagRef get tag reference
func (c *Client) GetTagRef(owner, repo, tag string) (*github.Reference, error) {
	resp, _, err := c.svc.Git.GetRef(c.ctx, owner, repo, "tags/"+tag)
	if err != nil {
		return nil, err
	}
	return resp, err
}
