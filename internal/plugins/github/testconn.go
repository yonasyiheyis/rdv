package github

import (
	"context"
	"fmt"
	"net/url"
	"os"

	ghapi "github.com/google/go-github/v57/github"
	"golang.org/x/oauth2"
)

func testToken(p *ghProfile) error {
	base := p.APIBase
	if base == "" {
		base = "https://api.github.com/"
	}

	ctx := context.Background()
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: p.Token})
	tc := oauth2.NewClient(ctx, ts)

	client := ghapi.NewClient(tc)
	if base != "https://api.github.com/" {
		// Set both API and upload URLs to your enterprise base
		var err error
		client, err = client.WithEnterpriseURLs(base, base)
		if err != nil {
			return err
		}
	}

	// Optional endpoint override for tests
	if ep := os.Getenv("GITHUB_API_URL"); ep != "" {
		if u, err := url.Parse(ep); err == nil {
			client.BaseURL = u
		}
	}

	u, _, err := client.Users.Get(ctx, "")
	if err != nil {
		return fmt.Errorf("GitHub API call failed: %w", err)
	}

	p.User = u.GetLogin()
	fmt.Printf("✅ GitHub token valid for user %s\n", p.User)
	return nil
}
