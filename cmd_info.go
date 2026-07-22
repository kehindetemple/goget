package main

import (
	"fmt"

	"github.com/kehindetemple/goget/internal/ghclient"
	"github.com/kehindetemple/goget/internal/storage"
)

// cmdInfo implements Feature 5 (Package Information).
func cmdInfo(name string) error {
	cfg, err := storage.LoadConfig()
	if err != nil {
		return err
	}
	gh := ghclient.New(cfg.GitHubToken)

	repo, err := resolveRepo(gh, name)
	if err != nil {
		return err
	}

	// Refetch full metadata directly, since search results omit some fields.
	full, err := gh.GetRepo(repo.Owner.Login, repo.Name)
	if err != nil {
		full = *repo // fall back to what we already have
	}

	release, err := gh.LatestRelease(full.Owner.Login, full.Name)
	if err != nil {
		release.TagName = "unavailable"
	}
	latest := release.TagName
	if latest == "" {
		latest = "no releases published"
	}

	license := "none"
	if full.License != nil && full.License.Name != "" {
		license = full.License.Name
	}

	desc := full.Description
	if desc == "" {
		desc = "(no description)"
	}

	fmt.Printf("Package:      %s\n", full.FullName)
	fmt.Printf("Description:  %s\n", desc)
	fmt.Printf("Stars:        %d\n", full.StargazersCount)
	fmt.Printf("License:      %s\n", license)
	fmt.Printf("Latest:       %s\n", latest)
	fmt.Printf("Repository:   %s\n", full.HTMLURL)
	fmt.Printf("Last Updated: %s\n", full.UpdatedAt.Format("2006-01-02"))
	fmt.Printf("Module path:  %s\n", moduleFor(&full))
	return nil
}
