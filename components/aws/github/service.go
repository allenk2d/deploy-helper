package github

import (
	"context"

	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
)

// IfaceGithub interface
//go:generate mockery --name IfaceGithub --output ../mocks
type IfaceGithub interface {
	GetReleaseURL(owner, repo, tag string) string
	GetReleaseByTag(owner, repo, tag string) (*github.RepositoryRelease, error)
	GetTagRef(owner, repo, tag string) (*github.Reference, error)
}

// Client struct
type Client struct {
	svc *github.Client
	ctx context.Context
}

// New getClient
func New(token string) *Client {
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)

	client := github.NewClient(tc)

	c := &Client{
		client,
		ctx,
	}
	return c
}
