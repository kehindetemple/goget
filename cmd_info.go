package main

import (
	"fmt"
	"strings"

	"github.com/kehindetemple/goget/internal/registry"
)

func cmdInfo(name string) error {
	pkg, _, err := resolvePackage(name, false)
	if err != nil {
		return err
	}
	verified := "No"
	if pkg.Verified {
		verified = "Yes"
	}
	deprecated := "No"
	if pkg.Deprecated {
		deprecated = "Yes"
	}
	latest := pkg.Latest
	if latest == "" {
		latest = "managed by Go modules"
	}
	license := pkg.License
	if license == "" {
		license = "not available"
	}

	fmt.Printf("%s\n\n", pkg.Name)
	fmt.Printf("Module:       %s\n", pkg.Module)
	if pkg.Package != "" {
		fmt.Printf("Package:      %s\n", pkg.Package)
	}
	fmt.Printf("Type:         %s\n", title(string(pkg.Type)))
	fmt.Printf("Install:      %s\n", pkg.InstallMethod)
	fmt.Printf("Latest:       %s\n", latest)
	fmt.Printf("License:      %s\n", license)
	fmt.Printf("Verified:     %s\n", verified)
	fmt.Printf("Deprecated:    %s\n", deprecated)
	fmt.Printf("Health:       %d/100\n", pkg.Health.Score)
	if len(pkg.Aliases) > 0 {
		fmt.Printf("Aliases:      %v\n", pkg.Aliases)
	}
	if pkg.Description != "" {
		fmt.Printf("Description:  %s\n", pkg.Description)
	}
	if pkg.Health.Score > 0 {
		fmt.Println()
		fmt.Printf("Maintenance:   %d\n", pkg.Health.Maintenance)
		fmt.Printf("Popularity:    %d\n", pkg.Health.Popularity)
		fmt.Printf("Activity:      %d\n", pkg.Health.Activity)
		fmt.Printf("Documentation: %d\n", pkg.Health.Documentation)
		fmt.Printf("Security:      %d\n", pkg.Health.Security)
	}
	if pkg.Type == registry.Library && pkg.Package == "" {
		fmt.Printf("\nImport path:   %s\n", pkg.Module)
	}
	return nil
}

func title(value string) string {
	return strings.ToUpper(value[:1]) + value[1:]
}
