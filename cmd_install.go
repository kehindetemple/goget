package main

import (
	"fmt"
	"strings"

	"github.com/kehindetemple/goget/internal/ghclient"
	"github.com/kehindetemple/goget/internal/installer"
	"github.com/kehindetemple/goget/internal/storage"
)

func moduleFor(r *ghclient.Repo) string {
	return "github.com/" + r.Owner.Login + "/" + r.Name
}

// cmdInstall keeps the original API while enabling the v3 resolver.
func cmdInstall(name, saveProfile string) error {
	return cmdInstallWithOptions(name, saveProfile, false)
}

func cmdInstallWithOptions(name, saveProfile string, offline bool) error {
	request := parsePackageRequest(name)
	pkg, version, err := resolvePackage(name, offline)
	if err != nil {
		return err
	}

	cache, err := storage.LoadCache()
	if err != nil {
		return fmt.Errorf("cache error: %w", err)
	}
	cacheEntry := storage.CacheEntry{
		Name:          pkg.Name,
		Module:        pkg.Module,
		Package:       pkg.Package,
		Type:          string(pkg.Type),
		InstallMethod: string(pkg.InstallMethod),
	}
	cache.SetEntry(request.Name, cacheEntry)
	cache.SetEntry(pkg.Name, cacheEntry)
	if err := storage.SaveCache(cache); err != nil {
		fmt.Println("warning: failed to update cache:", err)
	}

	friendlyVersion := version
	if friendlyVersion == "" {
		friendlyVersion = "latest"
	}
	fmt.Printf("Resolved %s -> %s\n", request.Name, pkg.Module)
	fmt.Printf("Type: %s | Install method: %s | Version: %s\n", pkg.Type, pkg.InstallMethod, friendlyVersion)
	if pkg.Verified {
		fmt.Println("Verified: yes")
	} else {
		fmt.Println("Verified: no (discovered fallback)")
	}
	install := installer.InstallPackage
	if offline {
		install = installer.InstallOffline
	}
	if err := install(pkg.Module, string(pkg.Type), friendlyVersion); err != nil {
		return fmt.Errorf("installation error: package resolved successfully, but installation failed: %w", err)
	}
	fmt.Printf("✔ %s installed successfully (%s)\n", pkg.Name, pkg.Module)

	if err := storage.AddHistoryEntry(storage.HistoryEntry{Package: pkg.Name, Module: pkg.Module, Type: string(pkg.Type)}); err != nil {
		fmt.Println("warning: failed to update history:", err)
	}

	if saveProfile != "" {
		if err := storage.SaveToProfile(saveProfile, pkg.Name, pkg.Module); err != nil {
			return fmt.Errorf("installed, but failed to save to profile %q: %w", saveProfile, err)
		}
		fmt.Printf("✔ saved %s to profile %q\n", pkg.Name, saveProfile)
	}
	return nil
}

// parseInstallArgs pulls flags out of the argument list, e.g.
// `goget gin --save api`.
func parseInstallArgs(args []string) (pkg string, saveProfile string, err error) {
	if len(args) == 0 {
		return "", "", fmt.Errorf("no package name given")
	}
	pkg = args[0]
	rest := args[1:]
	for i := 0; i < len(rest); i++ {
		if strings.EqualFold(rest[i], "--save") {
			if i+1 >= len(rest) {
				return "", "", fmt.Errorf("--save requires a profile name")
			}
			saveProfile = rest[i+1]
			i++
			continue
		}
		return "", "", fmt.Errorf("unknown option %q", rest[i])
	}
	return pkg, saveProfile, nil
}
