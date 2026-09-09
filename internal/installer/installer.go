// Package installer delegates dependency changes to the Go toolchain.
package installer

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Install is the GoGet 1.x compatibility entry point. Historically it meant
// installing a standalone command, so it retains that behavior.
func Install(modulePath string) error {
	return InstallPackage(modulePath, "command", "latest")
}

// InstallPackage chooses the safe Go operation from structured registry metadata.
func InstallPackage(modulePath, packageType, version string) error {
	return install(modulePath, packageType, version, false)
}

// InstallOffline keeps Go's module resolver from reaching the network. It can
// still succeed when the requested module is already in the local module cache.
func InstallOffline(modulePath, packageType, version string) error {
	return install(modulePath, packageType, version, true)
}

func install(modulePath, packageType, version string, offline bool) error {
	if modulePath == "" {
		return fmt.Errorf("module path cannot be empty")
	}
	if version == "" {
		version = "latest"
	}
	if packageType == "command" {
		return installCommand(modulePath, version, offline)
	}
	return installLibrary(modulePath, version, offline)
}

// InstallLibrary adds a dependency to the current Go project. It refuses to
// invent a module when the caller is outside a Go project.
func InstallLibrary(modulePath, version string) error {
	return installLibrary(modulePath, version, false)
}

func installLibrary(modulePath, version string, offline bool) error {
	if !InGoProject() {
		return fmt.Errorf("cannot add library %s outside a Go project; run this command from a directory containing go.mod", modulePath)
	}
	if version == "" {
		version = "latest"
	}
	return runGo("get", modulePath+"@"+version, "go get", offline)
}

// InstallCommand installs a standalone executable into Go's configured bin.
func InstallCommand(modulePath, version string) error {
	return installCommand(modulePath, version, false)
}

func installCommand(modulePath, version string, offline bool) error {
	if version == "" {
		version = "latest"
	}
	return runGo("install", modulePath+"@"+version, "go install", offline)
}

// InstallLegacy is kept for callers of GoGet 1.x.
func InstallLegacy(modulePath string) error {
	return InstallCommand(modulePath, "latest")
}

func runGo(command, target, label string, offline bool) error {
	cmd := exec.Command("go", command, target)
	if offline {
		cmd.Env = append(os.Environ(), "GOPROXY=off")
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s %s failed: %w", label, target, err)
	}
	return nil
}

func InGoProject() bool {
	workingDir, err := os.Getwd()
	if err != nil {
		return false
	}
	for {
		if _, err := os.Stat(filepath.Join(workingDir, "go.mod")); err == nil {
			return true
		}
		parent := filepath.Dir(workingDir)
		if parent == workingDir {
			return false
		}
		workingDir = parent
	}
}

func GoVersion() (string, error) {
	output, err := exec.Command("go", "version").CombinedOutput()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

func GoEnv(name string) (string, error) {
	output, err := exec.Command("go", "env", name).CombinedOutput()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}
