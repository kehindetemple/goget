// Package storage handles all local persistence for GoGet:
// config.json, cache.json, history.json and profiles.json,
// stored under ~/.config/goget/ as specified in the PRD.
package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const maxHistoryEntries = 50

// Dir returns (and creates if necessary) the goget config directory.
func Dir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("could not determine home directory: %w", err)
	}
	dir := filepath.Join(home, ".config", "goget")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("could not create config directory %s: %w", dir, err)
	}
	return dir, nil
}

func path(name string) (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, name), nil
}

func readJSON(name string, v interface{}) error {
	p, err := path(name)
	if err != nil {
		return err
	}
	data, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // caller sees the zero value
		}
		return err
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return nil
	}
	return json.Unmarshal(data, v)
}

func writeJSON(name string, v interface{}) error {
	p, err := path(name)
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, data, 0o600)
}

// ---------- Config ----------

// Config holds user-level settings, currently just the GitHub token.
type Config struct {
	GitHubToken string `json:"github_token,omitempty"`
}

func LoadConfig() (Config, error) {
	var c Config
	err := readJSON("config.json", &c)
	return c, err
}

func SaveConfig(c Config) error {
	return writeJSON("config.json", c)
}

// ---------- Cache ----------

// CacheEntry maps a lower-cased search query to a resolved Go module path.
type Cache map[string]string

func LoadCache() (Cache, error) {
	c := Cache{}
	if err := readJSON("cache.json", &c); err != nil {
		return nil, err
	}
	if c == nil {
		c = Cache{}
	}
	return c, nil
}

func SaveCache(c Cache) error {
	return writeJSON("cache.json", c)
}

func (c Cache) Get(query string) (string, bool) {
	v, ok := c[strings.ToLower(query)]
	return v, ok
}

func (c Cache) Set(query, modulePath string) {
	c[strings.ToLower(query)] = modulePath
}

// ---------- History ----------

// HistoryEntry records a single successful install.
type HistoryEntry struct {
	Package     string    `json:"package"`
	Module      string    `json:"module"`
	InstalledAt time.Time `json:"installed_at"`
}

func LoadHistory() ([]HistoryEntry, error) {
	var h []HistoryEntry
	err := readJSON("history.json", &h)
	return h, err
}

func SaveHistory(h []HistoryEntry) error {
	return writeJSON("history.json", h)
}

// AddHistory prepends a new entry (most recent first) and caps the list.
func AddHistory(pkg, module string) error {
	h, err := LoadHistory()
	if err != nil {
		return err
	}
	entry := HistoryEntry{Package: pkg, Module: module, InstalledAt: time.Now()}
	h = append([]HistoryEntry{entry}, h...)
	if len(h) > maxHistoryEntries {
		h = h[:maxHistoryEntries]
	}
	return SaveHistory(h)
}

// ---------- Profiles ----------

// Profile is a named, ordered collection of package name -> module path.
type Profile struct {
	Name     string            `json:"name"`
	Packages map[string]string `json:"packages"`
	// Order preserves insertion order for stable display/installation order.
	Order []string `json:"order"`
}

type Profiles map[string]*Profile

func LoadProfiles() (Profiles, error) {
	p := Profiles{}
	if err := readJSON("profiles.json", &p); err != nil {
		return nil, err
	}
	if p == nil {
		p = Profiles{}
	}
	return p, nil
}

func SaveProfiles(p Profiles) error {
	return writeJSON("profiles.json", p)
}

func CreateProfile(name string) error {
	profiles, err := LoadProfiles()
	if err != nil {
		return err
	}
	key := strings.ToLower(name)
	if _, exists := profiles[key]; exists {
		return fmt.Errorf("profile %q already exists", name)
	}
	profiles[key] = &Profile{Name: name, Packages: map[string]string{}}
	return SaveProfiles(profiles)
}

func GetProfile(name string) (*Profile, error) {
	profiles, err := LoadProfiles()
	if err != nil {
		return nil, err
	}
	p, ok := profiles[strings.ToLower(name)]
	if !ok {
		return nil, fmt.Errorf("profile %q does not exist", name)
	}
	return p, nil
}

func ListProfiles() ([]string, error) {
	profiles, err := LoadProfiles()
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(profiles))
	for _, p := range profiles {
		names = append(names, p.Name)
	}
	sort.Strings(names)
	return names, nil
}

// SaveToProfile adds/updates a package entry in the named profile,
// creating the profile if it doesn't already exist.
func SaveToProfile(profileName, pkgName, modulePath string) error {
	profiles, err := LoadProfiles()
	if err != nil {
		return err
	}
	key := strings.ToLower(profileName)
	p, ok := profiles[key]
	if !ok {
		p = &Profile{Name: profileName, Packages: map[string]string{}}
		profiles[key] = p
	}
	if _, existed := p.Packages[pkgName]; !existed {
		p.Order = append(p.Order, pkgName)
	}
	p.Packages[pkgName] = modulePath
	return SaveProfiles(profiles)
}

// RemoveFromProfile deletes a single package entry from a profile.
func RemoveFromProfile(profileName, pkgName string) error {
	profiles, err := LoadProfiles()
	if err != nil {
		return err
	}
	key := strings.ToLower(profileName)
	p, ok := profiles[key]
	if !ok {
		return fmt.Errorf("profile %q does not exist", profileName)
	}
	if _, ok := p.Packages[pkgName]; !ok {
		return fmt.Errorf("package %q not found in profile %q", pkgName, profileName)
	}
	delete(p.Packages, pkgName)
	for i, n := range p.Order {
		if n == pkgName {
			p.Order = append(p.Order[:i], p.Order[i+1:]...)
			break
		}
	}
	return SaveProfiles(profiles)
}

// DeleteProfile removes an entire profile.
func DeleteProfile(name string) error {
	profiles, err := LoadProfiles()
	if err != nil {
		return err
	}
	key := strings.ToLower(name)
	if _, ok := profiles[key]; !ok {
		return fmt.Errorf("profile %q does not exist", name)
	}
	delete(profiles, key)
	return SaveProfiles(profiles)
}

// OrderedPackages returns the profile's packages in insertion order.
func (p *Profile) OrderedPackages() []struct{ Name, Module string } {
	out := make([]struct{ Name, Module string }, 0, len(p.Packages))
	seen := map[string]bool{}
	for _, name := range p.Order {
		if mod, ok := p.Packages[name]; ok && !seen[name] {
			out = append(out, struct{ Name, Module string }{name, mod})
			seen[name] = true
		}
	}
	// Include any packages missing from Order (defensive, e.g. hand-edited file).
	for name, mod := range p.Packages {
		if !seen[name] {
			out = append(out, struct{ Name, Module string }{name, mod})
		}
	}
	return out
}
