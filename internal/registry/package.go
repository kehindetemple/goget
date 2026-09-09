// Package registry provides GoGet's local and remote package catalog.
package registry

import (
	"sort"
	"strings"
)

// PackageType describes whether a module should be added to a project or
// installed as a standalone executable.
type PackageType string

const (
	Library PackageType = "library"
	Command PackageType = "command"
)

// InstallMethod is intentionally constrained to Go's two supported flows.
type InstallMethod string

const (
	GoGet     InstallMethod = "go_get"
	GoInstall InstallMethod = "go_install"
)

type Health struct {
	Score         int `json:"score"`
	Maintenance   int `json:"maintenance"`
	Popularity    int `json:"popularity"`
	Activity      int `json:"activity"`
	Documentation int `json:"documentation"`
	Security      int `json:"security"`
}

// Package is the registry representation of a Go module.
type Package struct {
	Name          string        `json:"name"`
	Aliases       []string      `json:"aliases,omitempty"`
	Module        string        `json:"module"`
	Package       string        `json:"package,omitempty"`
	Type          PackageType   `json:"type"`
	InstallMethod InstallMethod `json:"install_method"`
	Verified      bool          `json:"verified"`
	Deprecated    bool          `json:"deprecated"`
	Archived      bool          `json:"archived,omitempty"`
	Description   string        `json:"description,omitempty"`
	Categories    []string      `json:"categories,omitempty"`
	Latest        string        `json:"latest,omitempty"`
	License       string        `json:"license,omitempty"`
	Stars         int           `json:"stars,omitempty"`
	Health        Health        `json:"health,omitempty"`
}

// Registry is an in-memory index. It has no architectural size limit; the
// built-in catalog and persisted discoveries are simply merged at startup.
type Registry struct {
	packages []Package
	byKey    map[string]int
}

func New(packages []Package) *Registry {
	r := &Registry{byKey: make(map[string]int)}
	for _, p := range packages {
		r.Add(p)
	}
	return r
}

// Load returns the built-in catalog plus packages discovered in prior runs.
func Load() *Registry {
	r := New(DefaultPackages())
	for _, p := range loadPersisted() {
		r.Add(p)
	}
	return r
}

func normalize(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func (r *Registry) Add(p Package) {
	if p.Name == "" || p.Module == "" {
		return
	}
	if p.Type == "" {
		p.Type = Library
	}
	if p.InstallMethod == "" {
		if p.Type == Command {
			p.InstallMethod = GoInstall
		} else {
			p.InstallMethod = GoGet
		}
	}
	if p.Health.Score == 0 {
		p.Health = calculateHealth(p)
	}
	key := normalize(p.Name)
	if index, ok := r.byKey[key]; ok {
		if r.packages[index].Verified && !p.Verified {
			return
		}
		r.packages[index] = p
		r.rebuildIndex()
		return
	}
	r.packages = append(r.packages, p)
	r.rebuildIndex()
}

func (r *Registry) rebuildIndex() {
	r.byKey = make(map[string]int, len(r.packages)*2)
	for i, p := range r.packages {
		r.byKey[normalize(p.Name)] = i
		r.byKey[normalize(p.Module)] = i
		for _, alias := range p.Aliases {
			r.byKey[normalize(alias)] = i
		}
	}
}

func (r *Registry) Resolve(query string) (Package, bool) {
	index, ok := r.byKey[normalize(query)]
	if !ok {
		return Package{}, false
	}
	return r.packages[index], true
}

func (r *Registry) All() []Package {
	out := append([]Package(nil), r.packages...)
	return out
}

type scored struct {
	p     Package
	score int
}

// Search ranks exact names, aliases, module paths, categories and descriptions
// in that order. Stable alphabetical ordering keeps terminal output reliable.
func (r *Registry) Search(query string) []Package {
	q := normalize(query)
	if q == "" {
		return nil
	}
	var matches []scored
	for _, p := range r.packages {
		score := matchScore(p, q)
		if score >= 0 {
			matches = append(matches, scored{p: p, score: score})
		}
	}
	sort.SliceStable(matches, func(i, j int) bool {
		if matches[i].score != matches[j].score {
			return matches[i].score < matches[j].score
		}
		if matches[i].p.Stars != matches[j].p.Stars {
			return matches[i].p.Stars > matches[j].p.Stars
		}
		return matches[i].p.Name < matches[j].p.Name
	})
	out := make([]Package, len(matches))
	for i, match := range matches {
		out[i] = match.p
	}
	return out
}

func matchScore(p Package, query string) int {
	name := normalize(p.Name)
	if name == query {
		return 0
	}
	for _, alias := range p.Aliases {
		if normalize(alias) == query {
			return 1
		}
	}
	if strings.HasPrefix(name, query) || strings.HasPrefix(normalize(p.Module), query) {
		return 2
	}
	for _, alias := range p.Aliases {
		if strings.Contains(normalize(alias), query) {
			return 3
		}
	}
	if strings.Contains(normalize(p.Module), query) {
		return 3
	}
	for _, category := range p.Categories {
		if strings.Contains(normalize(category), query) {
			return 4
		}
	}
	if strings.Contains(normalize(p.Description), query) {
		return 5
	}
	return -1
}

func calculateHealth(p Package) Health {
	base := 60
	if p.Verified {
		base += 12
	}
	if p.Stars > 10000 {
		base += 15
	} else if p.Stars > 1000 {
		base += 10
	} else if p.Stars > 100 {
		base += 5
	}
	if p.Description != "" {
		base += 5
	}
	if base > 99 {
		base = 99
	}
	return Health{Score: base, Maintenance: base, Popularity: base, Activity: base, Documentation: base, Security: base}
}
