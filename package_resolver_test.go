package main

import (
	"testing"

	"github.com/kehindetemple/goget/internal/registry"
)

func TestParsePackageRequest(t *testing.T) {
	request := parsePackageRequest("gin@v1.10.0")
	if request.Name != "gin" || request.Version != "v1.10.0" {
		t.Fatalf("unexpected request: %+v", request)
	}
	request = parsePackageRequest("github.com/gin-gonic/gin")
	if request.Name != "github.com/gin-gonic/gin" || request.Version != "latest" {
		t.Fatalf("unexpected module request: %+v", request)
	}
}

func TestExactModuleClassification(t *testing.T) {
	pkg := packageFromModule("github.com/air-verse/air/cmd/air")
	if pkg.Type != registry.Command || pkg.InstallMethod != registry.GoInstall {
		t.Fatalf("expected command classification: %+v", pkg)
	}
	pkg = packageFromModule("github.com/gohugoio/hugo")
	if pkg.Type != registry.Command || pkg.InstallMethod != registry.GoInstall {
		t.Fatalf("expected Hugo command classification: %+v", pkg)
	}
	pkg = packageFromModule("go.k6.io/k6/v2")
	if pkg.Type != registry.Command || pkg.InstallMethod != registry.GoInstall {
		t.Fatalf("expected k6 command classification: %+v", pkg)
	}
	pkg = packageFromModule("github.com/gin-gonic/gin")
	if pkg.Type != registry.Library || pkg.InstallMethod != registry.GoGet {
		t.Fatalf("expected library classification: %+v", pkg)
	}
}

func TestTypoSuggestionHandlesTransposition(t *testing.T) {
	pkg, ok := suggestPackage(registry.New(registry.DefaultPackages()), "gni")
	if !ok || pkg.Name != "gin" {
		t.Fatalf("expected gin suggestion, got %q", pkg.Name)
	}
}

func TestTypoSuggestionHandlesMisspelling(t *testing.T) {
	pkg, ok := suggestPackage(registry.New(registry.DefaultPackages()), "bycrpt")
	if !ok || pkg.Name != "bcrypt" {
		t.Fatalf("expected bcrypt suggestion, got %q", pkg.Name)
	}
}

func TestTypoSuggestionDoesNotMatchUnrelatedShortQuery(t *testing.T) {
	catalog := registry.New([]registry.Package{{Name: "ai", Module: "example.com/ai"}})
	if pkg, ok := suggestPackage(catalog, "k6"); ok {
		t.Fatalf("unexpected suggestion for k6: %q", pkg.Name)
	}
}

func TestRegistryResolvesHugoAndK6AsCommands(t *testing.T) {
	catalog := registry.New(registry.DefaultPackages())
	for name, module := range map[string]string{
		"hugo": "github.com/gohugoio/hugo",
		"k6":   "go.k6.io/k6/v2",
	} {
		pkg, ok := catalog.Resolve(name)
		if !ok {
			t.Fatalf("expected %q to resolve", name)
		}
		if pkg.Module != module || pkg.Type != registry.Command || pkg.InstallMethod != registry.GoInstall {
			t.Fatalf("unexpected metadata for %q: %+v", name, pkg)
		}
	}
}
