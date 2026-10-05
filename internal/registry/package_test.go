package registry

import "testing"

func TestResolveAliasAndInstallMethod(t *testing.T) {
	catalog := New(DefaultPackages())
	pkg, ok := catalog.Resolve("live-reload")
	if !ok {
		t.Fatal("expected live-reload alias to resolve")
	}
	if pkg.Name != "air" || pkg.Type != Command || pkg.InstallMethod != GoInstall {
		t.Fatalf("unexpected alias resolution: %+v", pkg)
	}
}

func TestSearchUsesCategories(t *testing.T) {
	catalog := New(DefaultPackages())
	results := catalog.Search("authentication")
	if len(results) == 0 {
		t.Fatal("expected category search results")
	}
	for _, pkg := range results {
		if pkg.Type == "" || pkg.Module == "" {
			t.Fatalf("incomplete registry package: %+v", pkg)
		}
	}
}

func TestCatalogHasLaunchScale(t *testing.T) {
	if got := len(DefaultPackages()); got < 1000 {
		t.Fatalf("catalog has %d packages; expected at least 1000", got)
	}
}

func TestResolveMongoDriverAliasesAndModulePath(t *testing.T) {
	catalog := New(DefaultPackages())
	for _, name := range []string{"mongo", "mongodb", "mongo-driver"} {
		pkg, ok := catalog.Resolve(name)
		if !ok {
			t.Fatalf("expected %q to resolve", name)
		}
		if pkg.Name != "mongo-go-driver" || pkg.Module != "go.mongodb.org/mongo-driver/v2" || pkg.Package != "go.mongodb.org/mongo-driver/v2/mongo" {
			t.Fatalf("unexpected MongoDB driver metadata for %q: %+v", name, pkg)
		}
	}
}

func TestCuratedAliasIsNotShadowedBySupplementalPackage(t *testing.T) {
	pkg, ok := New(DefaultPackages()).Resolve("cli")
	if !ok || pkg.Name != "urfave-cli" {
		t.Fatalf("cli resolved to %+v, %t; want urfave-cli", pkg, ok)
	}
}

func TestSupplementalCatalogPreservesVersionedModulePaths(t *testing.T) {
	pkg, ok := New(DefaultPackages()).Resolve("xsync")
	if !ok {
		t.Fatal("expected xsync to resolve")
	}
	if pkg.Module != "github.com/puzpuzpuz/xsync/v4" {
		t.Fatalf("xsync module path = %q; want versioned module path", pkg.Module)
	}
}
