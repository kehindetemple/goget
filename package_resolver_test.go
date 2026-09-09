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
