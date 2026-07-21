package main

import (
	"fmt"

	"goget/internal/installer"
	"goget/internal/storage"
	"goget/internal/ui"
)

// cmdProfileCreate implements "goget profile create <name>".
func cmdProfileCreate(name string) error {
	if err := storage.CreateProfile(name); err != nil {
		return err
	}
	fmt.Printf("✔ profile %q created\n", name)
	return nil
}

// cmdProfileList implements "goget profile list".
func cmdProfileList() error {
	names, err := storage.ListProfiles()
	if err != nil {
		return err
	}
	if len(names) == 0 {
		fmt.Println("No profiles yet. Create one with: goget profile create <name>")
		return nil
	}
	for _, n := range names {
		fmt.Println(n)
	}
	return nil
}

// cmdProfileShow implements "goget profile show <name>".
func cmdProfileShow(name string) error {
	p, err := storage.GetProfile(name)
	if err != nil {
		return err
	}
	pkgs := p.OrderedPackages()
	if len(pkgs) == 0 {
		fmt.Printf("Profile %q has no saved packages yet.\n", p.Name)
		return nil
	}
	fmt.Printf("Profile: %s\n\n", p.Name)
	for _, pkg := range pkgs {
		fmt.Printf("  %-20s %s\n", pkg.Name, pkg.Module)
	}
	return nil
}

// cmdProfileDelete implements "goget profile delete <name>" with a
// confirmation prompt, per the PRD's User Flow for this feature.
func cmdProfileDelete(name string) error {
	if _, err := storage.GetProfile(name); err != nil {
		return err
	}
	if !ui.Confirm(fmt.Sprintf("Delete profile %q? This cannot be undone.", name)) {
		fmt.Println("Cancelled.")
		return nil
	}
	if err := storage.DeleteProfile(name); err != nil {
		return err
	}
	fmt.Printf("✔ profile %q deleted\n", name)
	return nil
}

// cmdProfileRemove implements "goget profile remove <profile> <package>".
func cmdProfileRemove(profileName, pkgName string) error {
	if err := storage.RemoveFromProfile(profileName, pkgName); err != nil {
		return err
	}
	fmt.Printf("✔ removed %s from profile %q\n", pkgName, profileName)
	return nil
}

// cmdUse implements Feature 7's "goget use <profile>": installs every
// package saved in a profile, one by one, then prints a summary.
func cmdUse(profileName string) error {
	p, err := storage.GetProfile(profileName)
	if err != nil {
		return err
	}
	pkgs := p.OrderedPackages()
	if len(pkgs) == 0 {
		fmt.Printf("Profile %q has no packages to install.\n", profileName)
		return nil
	}

	var succeeded, failed []string
	for _, pkg := range pkgs {
		fmt.Printf("Installing %s (%s) ...\n", pkg.Name, pkg.Module)
		if err := installer.Install(pkg.Module); err != nil {
			fmt.Printf("  ✘ failed: %v\n", err)
			failed = append(failed, pkg.Name)
			continue
		}
		fmt.Printf("  ✔ %s installed\n", pkg.Name)
		succeeded = append(succeeded, pkg.Name)
		if err := storage.AddHistory(pkg.Name, pkg.Module); err != nil {
			fmt.Println("  warning: failed to update history:", err)
		}
	}

	fmt.Println()
	fmt.Printf("Summary: %d installed, %d failed\n", len(succeeded), len(failed))
	if len(failed) > 0 {
		fmt.Println("Failed packages:", failed)
	}
	return nil
}
