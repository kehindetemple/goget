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

// cmdInstall implements Feature 1 (Smart Package Installation) plus the
// cache short-circuit (Feature 6) and optional profile save (Feature 7).
func cmdInstall(name string, saveProfile string) error {
	cfg, err := storage.LoadConfig()
	if err != nil {
		return err
	}
	gh := ghclient.New(cfg.GitHubToken)

	cache, err := storage.LoadCache()
	if err != nil {
		return err
	}

	var modulePath string
	var pkgName string

	if cached, ok := cache.Get(name); ok {
		fmt.Printf("Found %s in local cache -> %s\n", name, cached)
		modulePath = cached
		pkgName = name
	} else {
		repo, err := resolveRepo(gh, name)
		if err != nil {
			return err
		}
		modulePath = moduleFor(repo)
		pkgName = repo.Name
		fmt.Printf("Resolved %s -> %s\n", name, modulePath)

		cache.Set(name, modulePath)
		cache.Set(pkgName, modulePath) // also cache under the real package name
		if err := storage.SaveCache(cache); err != nil {
			fmt.Println("warning: failed to update cache:", err)
		}
	}

	fmt.Printf("Installing %s ...\n", modulePath)
	if err := installer.Install(modulePath); err != nil {
		return err
	}
	fmt.Printf("✔ %s installed successfully (%s)\n", pkgName, modulePath)

	if err := storage.AddHistory(pkgName, modulePath); err != nil {
		fmt.Println("warning: failed to update history:", err)
	}

	if saveProfile != "" {
		if err := storage.SaveToProfile(saveProfile, pkgName, modulePath); err != nil {
			return fmt.Errorf("installed, but failed to save to profile %q: %w", saveProfile, err)
		}
		fmt.Printf("✔ saved %s to profile %q\n", pkgName, saveProfile)
	}
	return nil
}

// parseInstallArgs pulls a trailing `--save <profile>` flag out of the
// argument list, e.g. `goget gin --save api`.
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
	}
	return pkg, saveProfile, nil
}
