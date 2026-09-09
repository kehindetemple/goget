package main

import (
	"fmt"
	"strings"

	"github.com/kehindetemple/goget/internal/ghclient"
	"github.com/kehindetemple/goget/internal/registry"
	"github.com/kehindetemple/goget/internal/storage"
)

func cmdSearch(query string) error {
	return cmdSearchWithOptions(query, false)
}

func cmdSearchWithOptions(query string, offline bool) error {
	query = strings.TrimSpace(query)
	if query == "" {
		return fmt.Errorf("search query cannot be empty")
	}
	catalog := registry.Load()
	results := catalog.Search(query)
	if len(results) == 0 && !offline {
		if remoteURL := getenv("GOGET_REGISTRY_URL"); remoteURL != "" {
			remoteResults, err := registry.NewRemoteClient(remoteURL).Search(query)
			if err == nil {
				results = remoteResults
			}
		}
	}
	if len(results) == 0 && !offline {
		cfg, err := storage.LoadConfig()
		if err != nil {
			return err
		}
		repos, err := ghclient.New(cfg.GitHubToken).SearchRepos(query, 10)
		if err == nil {
			for _, repo := range repos {
				results = append(results, packageFromRepo(&repo))
			}
		}
	}
	if len(results) == 0 {
		return fmt.Errorf("no packages found for %q", query)
	}

	fmt.Printf("GoGet packages matching %q\n\n", query)
	for i, pkg := range results {
		if i == 20 {
			break
		}
		verified := "unverified"
		if pkg.Verified {
			verified = "verified"
		}
		fmt.Printf("%2d. %-20s %-8s %s\n", i+1, pkg.Name, string(pkg.Type), verified)
		fmt.Printf("    %s\n", pkg.Module)
		if pkg.Description != "" {
			fmt.Printf("    %s\n", pkg.Description)
		}
	}
	return nil
}
