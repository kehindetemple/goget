package main

import (
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/kehindetemple/goget/internal/fuzzy"
	"github.com/kehindetemple/goget/internal/ghclient"
	"github.com/kehindetemple/goget/internal/registry"
	"github.com/kehindetemple/goget/internal/storage"
)

type packageRequest struct {
	Name    string
	Version string
}

func parsePackageRequest(input string) packageRequest {
	input = strings.TrimSpace(input)
	request := packageRequest{Name: input, Version: "latest"}
	if at := strings.LastIndex(input, "@"); at > 0 {
		request.Name = input[:at]
		if input[at+1:] != "" {
			request.Version = input[at+1:]
		}
	}
	return request
}

func isModulePath(name string) bool {
	return strings.Contains(name, "/") && (strings.Contains(strings.Split(name, "/")[0], ".") || strings.HasPrefix(name, "golang.org/"))
}

func packageFromModule(module string) registry.Package {
	module = strings.TrimSpace(module)
	name := path.Base(module)
	if name == "" || name == "." || name == "/" {
		name = module
	}
	typeOfPackage := registry.Library
	if looksLikeCommand(module) {
		typeOfPackage = registry.Command
	}
	return registry.Package{
		Name:          name,
		Module:        module,
		Type:          typeOfPackage,
		InstallMethod: installMethodFor(typeOfPackage),
		Description:   "Resolved directly from a Go module path",
	}
}

func looksLikeCommand(module string) bool {
	parts := strings.Split(strings.ToLower(module), "/")
	for _, part := range parts {
		if part == "cmd" {
			return true
		}
	}
	known := map[string]bool{
		"air": true, "buf": true, "delve": true, "dlv": true, "goreleaser": true,
		"golangci-lint": true, "govulncheck": true, "mockgen": true, "sqlc": true,
		"staticcheck": true, "swag": true, "templ": true,
	}
	return known[parts[len(parts)-1]]
}

func installMethodFor(packageType registry.PackageType) registry.InstallMethod {
	if packageType == registry.Command {
		return registry.GoInstall
	}
	return registry.GoGet
}

func resolvePackage(input string, offline bool) (registry.Package, string, error) {
	request := parsePackageRequest(input)
	if request.Name == "" {
		return registry.Package{}, "", fmt.Errorf("resolution error: package name cannot be empty")
	}
	if isModulePath(request.Name) {
		return packageFromModule(request.Name), request.Version, nil
	}

	localCache, err := storage.LoadCache()
	if err != nil {
		return registry.Package{}, "", fmt.Errorf("cache error: %w", err)
	}
	catalog := registry.Load()
	if cached, ok := localCache.GetEntry(request.Name); ok {
		pkg := packageFromModule(cached.Module)
		if known, found := catalog.Resolve(cached.Module); found {
			pkg = known
		}
		if cached.Name != "" {
			pkg.Name = cached.Name
		}
		if cached.Package != "" {
			pkg.Package = cached.Package
		}
		if cached.Type != "" {
			pkg.Type = registry.PackageType(cached.Type)
			pkg.InstallMethod = installMethodFor(pkg.Type)
		}
		return pkg, request.Version, nil
	}
	if pkg, ok := catalog.Resolve(request.Name); ok {
		return pkg, request.Version, nil
	}
	if offline {
		if suggestion, ok := suggestPackage(catalog, request.Name); ok {
			return registry.Package{}, "", fmt.Errorf("resolution error: %q is not in the local cache; did you mean %q? (offline mode made no network requests)", request.Name, suggestion.Name)
		}
		return registry.Package{}, "", fmt.Errorf("resolution error: %q is not in the local cache or registry; offline mode made no network requests", request.Name)
	}

	if remoteURL := strings.TrimSpace(getenv("GOGET_REGISTRY_URL")); remoteURL != "" {
		remote := registry.NewRemoteClient(remoteURL)
		if pkg, remoteErr := remote.Resolve(request.Name); remoteErr == nil {
			catalog.Add(pkg)
			_ = registry.Persist(pkg)
			return pkg, request.Version, nil
		}
	}

	cfg, err := storage.LoadConfig()
	if err != nil {
		return registry.Package{}, "", err
	}
	repo, err := resolveRepo(ghclient.New(cfg.GitHubToken), request.Name)
	if err != nil {
		return registry.Package{}, "", fmt.Errorf("discovery error: %w", err)
	}
	pkg := packageFromRepo(repo)
	if err := registry.Persist(pkg); err != nil {
		fmt.Printf("warning: discovered package was not persisted: %v\n", err)
	}
	return pkg, request.Version, nil
}

func suggestPackage(catalog *registry.Registry, query string) (registry.Package, bool) {
	bestDistance := 1000
	var best registry.Package
	for _, pkg := range catalog.All() {
		distance := typoDistance(query, pkg.Name)
		if distance < bestDistance || (distance == bestDistance && betterSuggestion(pkg, best)) {
			bestDistance = distance
			best = pkg
		}
	}
	threshold := len([]rune(query))/2 + 1
	return best, best.Name != "" && bestDistance <= threshold
}

func typoDistance(a, b string) int {
	if len([]rune(a)) == len([]rune(b)) {
		ra, rb := []rune(strings.ToLower(a)), []rune(strings.ToLower(b))
		for i := 0; i+1 < len(ra); i++ {
			if ra[i] == rb[i+1] && ra[i+1] == rb[i] {
				copyOfA := append([]rune(nil), ra...)
				copyOfA[i], copyOfA[i+1] = copyOfA[i+1], copyOfA[i]
				if string(copyOfA) == string(rb) {
					return 1
				}
			}
		}
	}
	return fuzzy.Distance(a, b)
}

func betterSuggestion(candidate, current registry.Package) bool {
	if candidate.Verified != current.Verified {
		return candidate.Verified
	}
	return candidate.Stars > current.Stars
}

func packageFromRepo(repo *ghclient.Repo) registry.Package {
	packageType := registry.Library
	if looksLikeCommand(repo.Name) {
		packageType = registry.Command
	}
	return registry.Package{
		Name:          repo.Name,
		Module:        moduleFor(repo),
		Type:          packageType,
		InstallMethod: installMethodFor(packageType),
		Verified:      false,
		Archived:      repo.Archived,
		Description:   repo.Description,
		Latest:        "",
		License:       licenseName(repo),
		Stars:         repo.StargazersCount,
		Categories:    []string{"discovered"},
	}
}

func licenseName(repo *ghclient.Repo) string {
	if repo.License == nil {
		return ""
	}
	return repo.License.Name
}

func getenv(name string) string {
	// Kept behind a small function so resolution remains easy to exercise in
	// tests without introducing a configuration package.
	return strings.TrimSpace(os.Getenv(name))
}
