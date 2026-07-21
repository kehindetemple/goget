package main

import (
	"fmt"

	"goget/internal/fuzzy"
	"goget/internal/ghclient"
	"goget/internal/ui"
)

// resolveRepo implements the "Complete Search Flow" from the PRD:
//
//	GitHub Search -> Exact Match? -> yes: use it
//	                              -> no:  Owner Exists? -> yes: show owner's Go repos
//	                                                    -> no:  fuzzy "Did you mean" suggestions
//	Multiple relevant matches at any stage -> interactive selection menu.
func resolveRepo(gh *ghclient.Client, name string) (*ghclient.Repo, error) {
	results, err := gh.SearchRepos(name, 10)
	if err != nil {
		return nil, err
	}

	// Exact match(es) by repo name.
	var exact []ghclient.Repo
	for _, r := range results {
		if equalFold(r.Name, name) {
			exact = append(exact, r)
		}
	}
	switch len(exact) {
	case 1:
		return &exact[0], nil
	case 0:
		// fall through to the branches below
	default:
		return pickRepo("Multiple packages match that name:", exact)
	}

	// No exact match, but GitHub returned other relevant repos —
	// feature 4: interactive selection.
	if len(results) > 0 {
		return pickRepo(fmt.Sprintf("No exact match for %q. Here are the closest packages:", name), results)
	}

	// Nothing at all from the name search — is it a GitHub username/org?
	exists, err := gh.UserExists(name)
	if err != nil {
		return nil, err
	}
	if exists {
		repos, err := gh.UserRepos(name)
		if err != nil {
			return nil, err
		}
		if len(repos) == 0 {
			return nil, fmt.Errorf("%q is a GitHub user, but has no public Go repositories", name)
		}
		if len(repos) > 10 {
			repos = repos[:10]
		}
		return pickRepo(fmt.Sprintf("%q is a GitHub user. Here are their top Go repositories:", name), repos)
	}

	// Not a package, not a user — try a broader search and suggest the
	// closest matches by edit distance (typo detection).
	broad, err := gh.SearchReposBroad(name, 25)
	if err != nil {
		return nil, err
	}
	if len(broad) == 0 {
		return nil, fmt.Errorf("no packages found for %q — check the spelling and try again", name)
	}
	ranked := fuzzy.RankByName(name, broad, func(r ghclient.Repo) string { return r.Name })
	top := make([]ghclient.Repo, 0, 5)
	for i := 0; i < len(ranked) && i < 5; i++ {
		top = append(top, ranked[i].Item)
	}
	return pickRepo("Did you mean:", top)
}

// pickRepo shows a numbered menu of repos and returns the user's choice.
func pickRepo(prompt string, repos []ghclient.Repo) (*ghclient.Repo, error) {
	labels := make([]string, len(repos))
	for i, r := range repos {
		desc := r.Description
		if desc == "" {
			desc = "no description"
		}
		labels[i] = fmt.Sprintf("%-35s ★ %-6d %s", r.FullName, r.StargazersCount, desc)
	}
	idx, err := ui.SelectOption(prompt, labels)
	if err != nil {
		return nil, err
	}
	if idx < 0 {
		return nil, fmt.Errorf("cancelled")
	}
	return &repos[idx], nil
}

func equalFold(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		ca, cb := a[i], b[i]
		if 'A' <= ca && ca <= 'Z' {
			ca += 'a' - 'A'
		}
		if 'A' <= cb && cb <= 'Z' {
			cb += 'a' - 'A'
		}
		if ca != cb {
			return false
		}
	}
	return true
}
