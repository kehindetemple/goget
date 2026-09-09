package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/kehindetemple/goget/internal/installer"
	"github.com/kehindetemple/goget/internal/registry"
	"github.com/kehindetemple/goget/internal/ui"
)

func cmdUpdate(name string) error {
	pkg, _, err := resolvePackage(name, false)
	if err != nil {
		return err
	}
	return installer.InstallPackage(pkg.Module, string(pkg.Type), "latest")
}

func cmdOutdated() error {
	if !installer.InGoProject() {
		return fmt.Errorf("outdated requires a Go project with go.mod")
	}
	cmd := exec.Command("go", "list", "-m", "-u", "-json", "all")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("could not inspect project modules: %w", err)
	}
	decoder := json.NewDecoder(bufio.NewReader(strings.NewReader(string(output))))
	found := false
	fmt.Printf("%-40s %-14s %s\n", "Package", "Current", "Latest")
	for {
		var module struct {
			Path    string `json:"Path"`
			Version string `json:"Version"`
			Update  *struct {
				Version string `json:"Version"`
			} `json:"Update"`
		}
		if err := decoder.Decode(&module); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return err
		}
		if module.Update != nil && module.Update.Version != "" {
			found = true
			fmt.Printf("%-40s %-14s %s\n", module.Path, module.Version, module.Update.Version)
		}
	}
	if !found {
		fmt.Println("No outdated modules found.")
	}
	return nil
}

func cmdRemove(name string) error {
	pkg, _, err := resolvePackage(name, false)
	if err != nil {
		return err
	}
	if pkg.Type == registry.Command {
		return fmt.Errorf("GoGet does not remove globally installed commands; remove the binary from your Go bin directory")
	}
	if !installer.InGoProject() {
		return fmt.Errorf("cannot remove library %s outside a Go project", pkg.Module)
	}
	cmd := exec.Command("go", "get", pkg.Module+"@none")
	cmd.Stdout, cmd.Stderr, cmd.Stdin = os.Stdout, os.Stderr, os.Stdin
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("go get %s@none failed: %w", pkg.Module, err)
	}
	fmt.Printf("Removed %s from the current project.\n", pkg.Name)
	return nil
}

func cmdInit() error {
	if installer.InGoProject() {
		return fmt.Errorf("a Go module already exists in this directory or a parent directory")
	}
	choice, err := ui.SelectOption("What are you building?", []string{"API", "CLI", "Web application", "Library", "Basic Go project"})
	if err != nil {
		return err
	}
	if choice < 0 {
		return fmt.Errorf("cancelled")
	}
	moduleName, err := ui.Text("Module path (for example example.com/myapp): ")
	if err != nil {
		return err
	}
	if moduleName == "" {
		return fmt.Errorf("module path cannot be empty")
	}
	cmd := exec.Command("go", "mod", "init", moduleName)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("go mod init failed: %w", err)
	}
	recommendations := [][]string{
		{"gin", "validator", "zap"},
		{"cobra", "viper", "zap"},
		{"templ", "chi", "htmx"},
		{"cobra", "testify"},
		{"chi", "slog"},
	}
	fmt.Printf("Initialized %s. Suggested packages: %s\n", moduleName, strings.Join(recommendations[choice], ", "))
	return nil
}
