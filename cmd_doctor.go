package main

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/kehindetemple/goget/internal/installer"
	"github.com/kehindetemple/goget/internal/registry"
	"github.com/kehindetemple/goget/internal/storage"
)

func cmdDoctor() error {
	fmt.Println("GoGet Doctor")
	fmt.Println()
	check("Go installation", func() error {
		_, err := exec.LookPath("go")
		return err
	})
	check("Go version", func() error {
		_, err := installer.GoVersion()
		return err
	})
	check("GOPATH", func() error {
		value, err := installer.GoEnv("GOPATH")
		if err == nil && value == "" {
			return fmt.Errorf("GOPATH is empty")
		}
		return err
	})
	check("GOBIN", func() error {
		value, err := installer.GoEnv("GOBIN")
		if err != nil {
			return err
		}
		if value == "" {
			value, err = installer.GoEnv("GOPATH")
			if err != nil {
				return err
			}
			value = filepath.Join(value, "bin")
		}
		for _, directory := range filepath.SplitList(os.Getenv("PATH")) {
			if samePath(directory, value) {
				return nil
			}
		}
		return fmt.Errorf("%s is not on PATH", value)
	})
	check("Network", func() error {
		client := &http.Client{Timeout: 3 * time.Second}
		resp, err := client.Get("https://proxy.golang.org/")
		if err != nil {
			return err
		}
		resp.Body.Close()
		if resp.StatusCode >= 500 {
			return fmt.Errorf("proxy returned %s", resp.Status)
		}
		return nil
	})
	check("Registry", func() error {
		if len(registry.Load().All()) == 0 {
			return fmt.Errorf("catalog is empty")
		}
		return nil
	})
	check("Cache", func() error {
		_, err := storage.LoadCache()
		return err
	})
	check("Module system", func() error {
		_, err := exec.LookPath("go")
		return err
	})
	return nil
}

func check(label string, fn func() error) {
	if err := fn(); err != nil {
		fmt.Printf("%-20s x %s\n", label, err)
		return
	}
	fmt.Printf("%-20s ok\n", label)
}

func samePath(a, b string) bool {
	return strings.EqualFold(filepath.Clean(a), filepath.Clean(b))
}
