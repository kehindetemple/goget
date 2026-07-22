package main

import (
	"fmt"

	"github.com/kehindetemple/goget/internal/storage"
	"github.com/kehindetemple/goget/internal/ui"
)

// cmdHistory implements Feature 8 (Installation History).
func cmdHistory() error {
	h, err := storage.LoadHistory()
	if err != nil {
		return err
	}
	if len(h) == 0 {
		fmt.Println("No installs recorded yet.")
		return nil
	}
	for _, e := range h {
		fmt.Printf("%-20s %-35s %s\n", e.Package, e.Module, e.InstalledAt.Format("2006-01-02 15:04"))
	}
	return nil
}

// cmdLogin implements Feature 9 (GitHub Authentication).
func cmdLogin() error {
	fmt.Println("Create a GitHub Personal Access Token (no scopes needed for public repo search)")
	fmt.Println("at: https://github.com/settings/tokens")
	fmt.Println()
	fmt.Println("Note: the token will be visible as you type it in this terminal.")
	token, err := ui.Text("Paste your token: ")
	if err != nil {
		return err
	}
	if token == "" {
		return fmt.Errorf("no token entered, aborting")
	}
	cfg, err := storage.LoadConfig()
	if err != nil {
		return err
	}
	cfg.GitHubToken = token
	if err := storage.SaveConfig(cfg); err != nil {
		return err
	}
	fmt.Println("✔ token saved. GitHub API rate limits are now increased for this machine.")
	return nil
}
