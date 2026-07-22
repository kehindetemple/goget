package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/kehindetemple/goget/internal/storage"
	"github.com/kehindetemple/goget/internal/ui"
)

// ---------------------------------------------------------------------------
// Gist API types
// ---------------------------------------------------------------------------

type gistFile struct {
	Content string `json:"content"`
}

type gistCreateRequest struct {
	Description string              `json:"description"`
	Public      bool                `json:"public"`
	Files       map[string]gistFile `json:"files"`
}

type gistResponse struct {
	ID      string                 `json:"id"`
	HTMLURL string                 `json:"html_url"`
	Files   map[string]gistFileRes `json:"files"`
}

type gistFileRes struct {
	Filename string `json:"filename"`
	RawURL   string `json:"raw_url"`
	Content  string `json:"content"`
}

// ---------------------------------------------------------------------------
// cmdProfileShare — uploads profile to a GitHub Gist
// ---------------------------------------------------------------------------

// cmdProfileShare implements "goget profile share <name>".
// It uploads the profile as a public GitHub Gist and prints the share link.
// A GitHub token is required (run `goget login` first).
func cmdProfileShare(name string) error {
	cfg, err := storage.LoadConfig()
	if err != nil {
		return err
	}
	if cfg.GitHubToken == "" {
		return fmt.Errorf("a GitHub token is required to share profiles.\nRun: goget login")
	}

	p, err := storage.GetProfile(name)
	if err != nil {
		return err
	}

	pkgs := p.OrderedPackages()
	if len(pkgs) == 0 {
		return fmt.Errorf("profile %q has no packages — add some first with: goget <pkg> --save %s", name, name)
	}

	// Marshal the profile to pretty JSON.
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to encode profile: %w", err)
	}

	filename := "goget-profile-" + strings.ToLower(name) + ".json"

	payload := gistCreateRequest{
		Description: fmt.Sprintf("GoGet profile: %s (shared %s)", name, time.Now().Format("2006-01-02")),
		Public:      true,
		Files: map[string]gistFile{
			filename: {Content: string(data)},
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, "https://api.github.com/gists", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+cfg.GitHubToken)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "goget-cli")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to upload profile: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("GitHub token is invalid or expired — run: goget login")
	}
	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var gist gistResponse
	if err := json.NewDecoder(resp.Body).Decode(&gist); err != nil {
		return fmt.Errorf("failed to parse GitHub response: %w", err)
	}

	fmt.Printf("✔ Profile %q shared successfully!\n\n", name)
	fmt.Printf("  Gist URL:  %s\n", gist.HTMLURL)
	fmt.Printf("  Gist ID:   %s\n\n", gist.ID)
	fmt.Println("Share this command with other developers:")
	fmt.Printf("\n  goget profile download %s\n\n", gist.ID)
	fmt.Println("They can then install all your packages with:")
	fmt.Printf("\n  goget use %s\n", name)

	return nil
}

// ---------------------------------------------------------------------------
// cmdProfileDownload — downloads a profile from a GitHub Gist
// ---------------------------------------------------------------------------

// cmdProfileDownload implements "goget profile download <gistID>".
// It fetches the profile JSON from a public GitHub Gist and saves it locally.
func cmdProfileDownload(gistID string) error {
	// Strip full URLs down to just the Gist ID.
	// e.g. https://gist.github.com/user/abc123 -> abc123
	gistID = strings.TrimSpace(gistID)
	if strings.Contains(gistID, "/") {
		parts := strings.Split(gistID, "/")
		gistID = parts[len(parts)-1]
	}

	cfg, err := storage.LoadConfig()
	if err != nil {
		return err
	}

	fmt.Printf("Fetching profile from Gist %s ...\n", gistID)

	req, err := http.NewRequest(http.MethodGet, "https://api.github.com/gists/"+gistID, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "goget-cli")
	if cfg.GitHubToken != "" {
		req.Header.Set("Authorization", "Bearer "+cfg.GitHubToken)
	}

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to fetch Gist: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("Gist %q not found — check the ID and try again", gistID)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var gist gistResponse
	if err := json.NewDecoder(resp.Body).Decode(&gist); err != nil {
		return fmt.Errorf("failed to parse Gist response: %w", err)
	}

	// Find the goget profile file inside the Gist.
	var profileContent string
	for filename, f := range gist.Files {
		if strings.HasPrefix(filename, "goget-profile-") && strings.HasSuffix(filename, ".json") {
			content := f.Content
			// If content is empty, fetch from RawURL.
			if content == "" && f.RawURL != "" {
				content, err = fetchRaw(f.RawURL, cfg.GitHubToken)
				if err != nil {
					return fmt.Errorf("failed to fetch profile content: %w", err)
				}
			}
			profileContent = content
			break
		}
	}

	if profileContent == "" {
		return fmt.Errorf("no GoGet profile found in Gist %q — make sure it was shared with 'goget profile share'", gistID)
	}

	// Parse the profile.
	var p storage.Profile
	if err := json.Unmarshal([]byte(profileContent), &p); err != nil {
		return fmt.Errorf("invalid profile data in Gist: %w", err)
	}
	if p.Name == "" {
		return fmt.Errorf("profile in Gist is missing a name")
	}

	// Check if profile already exists locally.
	profiles, err := storage.LoadProfiles()
	if err != nil {
		return err
	}
	key := strings.ToLower(p.Name)
	if _, exists := profiles[key]; exists {
		if !ui.Confirm(fmt.Sprintf("Profile %q already exists locally. Overwrite?", p.Name)) {
			fmt.Println("Cancelled.")
			return nil
		}
	}

	profiles[key] = &p
	if err := storage.SaveProfiles(profiles); err != nil {
		return err
	}

	pkgs := p.OrderedPackages()
	fmt.Printf("✔ Profile %q downloaded (%d packages)\n\n", p.Name, len(pkgs))
	for _, pkg := range pkgs {
		fmt.Printf("  %-20s %s\n", pkg.Name, pkg.Module)
	}
	fmt.Printf("\nInstall all packages with:\n\n  goget use %s\n", p.Name)
	return nil
}

// ---------------------------------------------------------------------------
// cmdProfileExport — export profile to a local JSON file
// ---------------------------------------------------------------------------

// cmdProfileExport implements "goget profile export <name>".
// Saves the profile as a JSON file in the current directory.
func cmdProfileExport(name string) error {
	p, err := storage.GetProfile(name)
	if err != nil {
		return err
	}

	filename := "goget-profile-" + strings.ToLower(name) + ".json"
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}

	if err := os.WriteFile(filename, data, 0o644); err != nil {
		return fmt.Errorf("could not write file: %w", err)
	}

	fmt.Printf("✔ Profile %q exported to %s\n", name, filename)
	fmt.Println("\nShare this file with others. They can import it with:")
	fmt.Printf("\n  goget profile import %s\n", filename)
	return nil
}

// ---------------------------------------------------------------------------
// cmdProfileImport — import profile from a local JSON file
// ---------------------------------------------------------------------------

// cmdProfileImport implements "goget profile import <file>".
func cmdProfileImport(filename string) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("could not read file %q: %w", filename, err)
	}

	var p storage.Profile
	if err := json.Unmarshal(data, &p); err != nil {
		return fmt.Errorf("invalid profile file: %w", err)
	}
	if p.Name == "" {
		return fmt.Errorf("profile file is missing a name")
	}

	profiles, err := storage.LoadProfiles()
	if err != nil {
		return err
	}

	key := strings.ToLower(p.Name)
	if _, exists := profiles[key]; exists {
		if !ui.Confirm(fmt.Sprintf("Profile %q already exists locally. Overwrite?", p.Name)) {
			fmt.Println("Cancelled.")
			return nil
		}
	}

	profiles[key] = &p
	if err := storage.SaveProfiles(profiles); err != nil {
		return err
	}

	pkgs := p.OrderedPackages()
	fmt.Printf("✔ Profile %q imported (%d packages)\n\n", p.Name, len(pkgs))
	for _, pkg := range pkgs {
		fmt.Printf("  %-20s %s\n", pkg.Name, pkg.Module)
	}
	fmt.Printf("\nInstall all packages with:\n\n  goget use %s\n", p.Name)
	return nil
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func fetchRaw(url, token string) (string, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "goget-cli")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	return string(b), err
}

// ---------------------------------------------------------------------------
// Registry configuration
// ---------------------------------------------------------------------------

type RegistryConfig struct {
	RepoURL string `json:"repo_url,omitempty"`
}

func loadRegistryConfig() (RegistryConfig, error) {
	// For now, we'll store registry info in a separate file
	var reg RegistryConfig
	p, err := registryConfigPath()
	if err != nil {
		return reg, err
	}
	data, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			return reg, nil
		}
		return reg, err
	}
	json.Unmarshal(data, &reg)
	return reg, nil
}

func saveRegistryConfig(reg RegistryConfig) error {
	p, err := registryConfigPath()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(reg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, data, 0o600)
}

func registryConfigPath() (string, error) {
	dir, err := storage.Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "registry.json"), nil
}

// dispatchRegistry handles registry subcommands
func dispatchRegistry(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: goget registry <set|show>")
	}
	switch args[0] {
	case "set":
		if len(args) < 2 {
			return fmt.Errorf("usage: goget registry set <repo-url>")
		}
		return cmdRegistrySet(args[1])
	case "show":
		return cmdRegistryShow()
	default:
		return fmt.Errorf("unknown registry subcommand %q", args[0])
	}
}

func cmdRegistrySet(repoURL string) error {
	repoURL = strings.TrimSpace(repoURL)
	if repoURL == "" {
		return fmt.Errorf("repo URL cannot be empty")
	}
	reg := RegistryConfig{RepoURL: repoURL}
	if err := saveRegistryConfig(reg); err != nil {
		return err
	}
	fmt.Printf("✔ Registry set to: %s\n", repoURL)
	fmt.Println("\nNow you can publish profiles with:")
	fmt.Println("  goget profile publish <name>")
	return nil
}

func cmdRegistryShow() error {
	reg, err := loadRegistryConfig()
	if err != nil {
		return err
	}
	if reg.RepoURL == "" {
		fmt.Println("No registry configured. Set one with:")
		fmt.Println("  goget registry set <repo-url>")
		fmt.Println("\nExample:")
		fmt.Println("  goget registry set github.com/yourname/goget-profiles")
		return nil
	}
	fmt.Printf("Registry: %s\n", reg.RepoURL)
	return nil
}

// ---------------------------------------------------------------------------
// cmdProfilePublish — publish a profile to your GitHub repo
// ---------------------------------------------------------------------------

func cmdProfilePublish(name string) error {
	cfg, err := storage.LoadConfig()
	if err != nil {
		return err
	}
	if cfg.GitHubToken == "" {
		return fmt.Errorf("a GitHub token is required to publish profiles.\nRun: goget login")
	}

	reg, err := loadRegistryConfig()
	if err != nil {
		return err
	}
	if reg.RepoURL == "" {
		return fmt.Errorf("no registry configured.\nRun: goget registry set <repo-url>")
	}

	p, err := storage.GetProfile(name)
	if err != nil {
		return err
	}

	pkgs := p.OrderedPackages()
	if len(pkgs) == 0 {
		return fmt.Errorf("profile %q has no packages", name)
	}

	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}

	// Parse repo URL to get owner and repo
	parts := strings.Split(strings.TrimPrefix(reg.RepoURL, "https://"), "/")
	if len(parts) < 2 {
		return fmt.Errorf("invalid repo URL format. Expected: github.com/owner/repo")
	}
	owner, repo := parts[len(parts)-2], parts[len(parts)-1]

	filename := name + ".json"
	path := "profiles/" + filename

	fmt.Printf("Publishing profile %q to %s ...\n", name, reg.RepoURL)

	// Use GitHub API to create/update file
	err = createOrUpdateGitHubFile(owner, repo, path, string(data), cfg.GitHubToken)
	if err != nil {
		return err
	}

	fmt.Printf("✔ Profile %q published to %s\n\n", name, reg.RepoURL)
	fmt.Println("Others can download it with:")
	fmt.Printf("  goget profile fetch %s %s\n", owner, name)
	return nil
}

// ---------------------------------------------------------------------------
// cmdProfileFetch — fetch a profile from another user's repo
// ---------------------------------------------------------------------------

func cmdProfileFetch(username, profilename string) error {
	// Assume the repo is at github.com/username/goget-profiles
	owner := username
	repo := "goget-profiles"
	path := "profiles/" + profilename + ".json"

	cfg, err := storage.LoadConfig()
	if err != nil {
		return err
	}

	fmt.Printf("Fetching profile %q from %s/%s ...\n", profilename, owner, repo)

	// Use GitHub raw content URL
	rawURL := fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/main/%s", owner, repo, path)

	profileContent, err := fetchRaw(rawURL, cfg.GitHubToken)
	if err != nil {
		return fmt.Errorf("failed to fetch profile: %w", err)
	}

	if strings.Contains(profileContent, "404") || profileContent == "" {
		return fmt.Errorf("profile %q not found in %s/%s", profilename, owner, repo)
	}

	var p storage.Profile
	if err := json.Unmarshal([]byte(profileContent), &p); err != nil {
		return fmt.Errorf("invalid profile data: %w", err)
	}
	if p.Name == "" {
		return fmt.Errorf("profile is missing a name")
	}

	profiles, err := storage.LoadProfiles()
	if err != nil {
		return err
	}

	key := strings.ToLower(p.Name)
	if _, exists := profiles[key]; exists {
		if !ui.Confirm(fmt.Sprintf("Profile %q already exists locally. Overwrite?", p.Name)) {
			fmt.Println("Cancelled.")
			return nil
		}
	}

	profiles[key] = &p
	if err := storage.SaveProfiles(profiles); err != nil {
		return err
	}

	pkgs := p.OrderedPackages()
	fmt.Printf("✔ Profile %q fetched (%d packages)\n\n", p.Name, len(pkgs))
	for _, pkg := range pkgs {
		fmt.Printf("  %-20s %s\n", pkg.Name, pkg.Module)
	}
	fmt.Printf("\nInstall all packages with:\n\n  goget use %s\n", p.Name)
	return nil
}

// ---------------------------------------------------------------------------
// GitHub file operations
// ---------------------------------------------------------------------------

type githubFileContent struct {
	Message   string      `json:"message"`
	Content   string      `json:"content"`
	SHA       string      `json:"sha,omitempty"`
	Committer interface{} `json:"committer"`
}

func createOrUpdateGitHubFile(owner, repo, path, content, token string) error {
	import64 := func(s string) string {
		return base64Encode([]byte(s))
	}

	// First, try to get existing file SHA
	getURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/contents/%s", owner, repo, path)
	req, _ := http.NewRequest(http.MethodGet, getURL, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("User-Agent", "goget-cli")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, _ := client.Do(req)
	var sha string
	if resp != nil && resp.StatusCode == 200 {
		var existing map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&existing)
		if s, ok := existing["sha"].(string); ok {
			sha = s
		}
		resp.Body.Close()
	}

	// Create or update file
	payload := map[string]interface{}{
		"message": "Add profile: " + path,
		"content": import64(content),
	}
	if sha != "" {
		payload["sha"] = sha
	}

	body, _ := json.Marshal(payload)
	putURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/contents/%s", owner, repo, path)
	req, _ = http.NewRequest(http.MethodPut, putURL, bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("User-Agent", "goget-cli")
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to publish: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("GitHub API returned status %d: %s", resp.StatusCode, string(body))
	}
	return nil
}

func base64Encode(data []byte) string {
	result := ""
	chars := "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"
	for i := 0; i < len(data); i += 3 {
		b1 := data[i]
		var b2, b3 byte
		var bits int
		if i+1 < len(data) {
			b2 = data[i+1]
			bits = int(b1)<<16 | int(b2)<<8
			if i+2 < len(data) {
				b3 = data[i+2]
				bits |= int(b3)
				result += string(chars[bits>>18&63])
				result += string(chars[bits>>12&63])
				result += string(chars[bits>>6&63])
				result += string(chars[bits&63])
			} else {
				bits >>= 8
				result += string(chars[bits>>12&63])
				result += string(chars[bits>>6&63])
				result += string(chars[bits&63])
				result += "="
			}
		} else {
			bits = int(b1) << 16
			result += string(chars[bits>>18&63])
			result += string(chars[bits>>12&63])
			result += "=="
		}
	}
	return result
}
