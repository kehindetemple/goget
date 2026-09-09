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
	if got := len(DefaultPackages()); got < 500 {
		t.Fatalf("catalog has %d packages; expected at least 500", got)
	}
}
