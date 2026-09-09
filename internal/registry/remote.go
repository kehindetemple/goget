package registry

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var ErrNotFound = errors.New("registry package not found")

// RemoteClient is optional. GOGET_REGISTRY_URL can point at a service that
// implements GET /resolve/:name and GET /search?q=... .
type RemoteClient struct {
	BaseURL    string
	HTTPClient *http.Client
}

func NewRemoteClient(baseURL string) *RemoteClient {
	return &RemoteClient{BaseURL: strings.TrimRight(baseURL, "/"), HTTPClient: &http.Client{Timeout: 10 * time.Second}}
}

func (c *RemoteClient) Resolve(name string) (Package, error) {
	var p Package
	if err := c.get("/resolve/"+url.PathEscape(name), &p); err != nil {
		return Package{}, err
	}
	if p.Name == "" || p.Module == "" {
		return Package{}, fmt.Errorf("registry returned incomplete metadata")
	}
	return p, nil
}

func (c *RemoteClient) Search(query string) ([]Package, error) {
	var response struct {
		Packages []Package `json:"packages"`
	}
	if err := c.get("/search?q="+url.QueryEscape(query), &response); err != nil {
		return nil, err
	}
	return response.Packages, nil
}

func (c *RemoteClient) get(path string, out interface{}) error {
	if c == nil || c.BaseURL == "" {
		return ErrNotFound
	}
	req, err := http.NewRequest(http.MethodGet, c.BaseURL+path, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("registry request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return ErrNotFound
	}
	if resp.StatusCode >= 300 {
		return fmt.Errorf("registry returned status %d", resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func loadPersisted() []Package {
	path, err := persistedPath()
	if err != nil {
		return nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var packages []Package
	if json.Unmarshal(data, &packages) != nil {
		return nil
	}
	return packages
}

// Persist stores a discovered package separately from the shipped catalog.
func Persist(p Package) error {
	path, err := persistedPath()
	if err != nil {
		return err
	}
	packages := loadPersisted()
	replaced := false
	for i, existing := range packages {
		if strings.EqualFold(existing.Name, p.Name) || strings.EqualFold(existing.Module, p.Module) {
			packages[i] = p
			replaced = true
			break
		}
	}
	if !replaced {
		packages = append(packages, p)
	}
	data, err := json.MarshalIndent(packages, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

func persistedPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "goget", "package-registry.json"), nil
}

func readBody(resp *http.Response) ([]byte, error) {
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}
