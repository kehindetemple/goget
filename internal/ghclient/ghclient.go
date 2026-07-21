// Package ghclient is a small GitHub REST API v3 client covering exactly
// what GoGet needs: repository search, owner/user lookup, repo metadata
// and latest-release lookup. No external dependencies.
package ghclient

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

const (
	apiBase   = "https://api.github.com"
	userAgent = "goget-cli"
)

// Client talks to the GitHub API, optionally authenticated with a
// Personal Access Token to raise rate limits.
type Client struct {
	Token      string
	HTTPClient *http.Client
}

func New(token string) *Client {
	return &Client{
		Token:      token,
		HTTPClient: &http.Client{Timeout: 15 * time.Second},
	}
}

// Repo is the subset of GitHub's repository object GoGet cares about.
type Repo struct {
	FullName        string    `json:"full_name"`
	Name            string    `json:"name"`
	Owner           RepoOwner `json:"owner"`
	Description     string    `json:"description"`
	StargazersCount int       `json:"stargazers_count"`
	HTMLURL         string    `json:"html_url"`
	License         *License  `json:"license"`
	UpdatedAt       time.Time `json:"updated_at"`
	Language        string    `json:"language"`
	Fork            bool      `json:"fork"`
	Archived        bool      `json:"archived"`
}

type RepoOwner struct {
	Login string `json:"login"`
}

type License struct {
	Name string `json:"name"`
	SPDX string `json:"spdx_id"`
}

type searchResponse struct {
	TotalCount int    `json:"total_count"`
	Items      []Repo `json:"items"`
}

type Release struct {
	TagName     string    `json:"tag_name"`
	Name        string    `json:"name"`
	PublishedAt time.Time `json:"published_at"`
	HTMLURL     string    `json:"html_url"`
}

func (c *Client) do(req *http.Request, out interface{}) error {
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("request to GitHub failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return errNotFound
	}
	if resp.StatusCode == http.StatusForbidden {
		return fmt.Errorf("GitHub API rate limit hit — run 'goget login' to add a personal access token and raise your limit")
	}
	if resp.StatusCode >= 300 {
		return fmt.Errorf("GitHub API returned status %d for %s", resp.StatusCode, req.URL.String())
	}
	if out == nil {
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

var errNotFound = fmt.Errorf("not found")

// IsNotFound reports whether err represents a GitHub 404.
func IsNotFound(err error) bool {
	return err == errNotFound
}

// SearchRepos searches GitHub for Go repositories matching query,
// sorted by stars descending. Returns up to `limit` results.
func (c *Client) SearchRepos(query string, limit int) ([]Repo, error) {
	q := url.Values{}
	q.Set("q", fmt.Sprintf("%s in:name language:Go", query))
	q.Set("sort", "stars")
	q.Set("order", "desc")
	q.Set("per_page", fmt.Sprintf("%d", limit))

	reqURL := apiBase + "/search/repositories?" + q.Encode()
	req, err := http.NewRequest(http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}
	var sr searchResponse
	if err := c.do(req, &sr); err != nil {
		return nil, err
	}
	// Filter out forks and archived repos — not useful as install targets.
	filtered := make([]Repo, 0, len(sr.Items))
	for _, r := range sr.Items {
		if !r.Fork && !r.Archived {
			filtered = append(filtered, r)
		}
	}
	return filtered, nil
}

// SearchReposBroad performs a looser full-text search (no `in:name`
// restriction), useful for surfacing fuzzy "did you mean" suggestions
// when an exact/substring name search comes back empty.
func (c *Client) SearchReposBroad(query string, limit int) ([]Repo, error) {
	q := url.Values{}
	q.Set("q", fmt.Sprintf("%s language:Go", query))
	q.Set("sort", "stars")
	q.Set("order", "desc")
	q.Set("per_page", fmt.Sprintf("%d", limit))

	reqURL := apiBase + "/search/repositories?" + q.Encode()
	req, err := http.NewRequest(http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}
	var sr searchResponse
	if err := c.do(req, &sr); err != nil {
		return nil, err
	}
	filtered := make([]Repo, 0, len(sr.Items))
	for _, r := range sr.Items {
		if !r.Fork && !r.Archived {
			filtered = append(filtered, r)
		}
	}
	return filtered, nil
}

// UserExists checks whether a GitHub user or organization with this
// login exists.
func (c *Client) UserExists(login string) (bool, error) {
	req, err := http.NewRequest(http.MethodGet, apiBase+"/users/"+url.PathEscape(login), nil)
	if err != nil {
		return false, err
	}
	err = c.do(req, nil)
	if err != nil {
		if IsNotFound(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// UserRepos fetches a user's public repositories, sorted by stars,
// filtered to non-fork Go repositories.
func (c *Client) UserRepos(login string) ([]Repo, error) {
	q := url.Values{}
	q.Set("sort", "updated")
	q.Set("per_page", "100")
	reqURL := apiBase + "/users/" + url.PathEscape(login) + "/repos?" + q.Encode()

	req, err := http.NewRequest(http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}
	var repos []Repo
	if err := c.do(req, &repos); err != nil {
		return nil, err
	}
	goRepos := make([]Repo, 0, len(repos))
	for _, r := range repos {
		if r.Language == "Go" && !r.Fork && !r.Archived {
			goRepos = append(goRepos, r)
		}
	}
	// Sort by stars desc.
	for i := 0; i < len(goRepos); i++ {
		for j := i + 1; j < len(goRepos); j++ {
			if goRepos[j].StargazersCount > goRepos[i].StargazersCount {
				goRepos[i], goRepos[j] = goRepos[j], goRepos[i]
			}
		}
	}
	return goRepos, nil
}

// GetRepo fetches full metadata for a specific owner/repo.
func (c *Client) GetRepo(owner, name string) (Repo, error) {
	req, err := http.NewRequest(http.MethodGet, apiBase+"/repos/"+url.PathEscape(owner)+"/"+url.PathEscape(name), nil)
	if err != nil {
		return Repo{}, err
	}
	var r Repo
	if err := c.do(req, &r); err != nil {
		return Repo{}, err
	}
	return r, nil
}

// LatestRelease fetches the latest release for owner/repo. Returns
// (Release{}, nil) with an empty TagName if the repo has no releases.
func (c *Client) LatestRelease(owner, name string) (Release, error) {
	req, err := http.NewRequest(http.MethodGet, apiBase+"/repos/"+url.PathEscape(owner)+"/"+url.PathEscape(name)+"/releases/latest", nil)
	if err != nil {
		return Release{}, err
	}
	var rel Release
	err = c.do(req, &rel)
	if err != nil {
		if IsNotFound(err) {
			return Release{}, nil
		}
		return Release{}, err
	}
	return rel, nil
}
