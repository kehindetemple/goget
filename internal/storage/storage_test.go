package storage

import (
	"encoding/json"
	"testing"
)

func TestCacheReadsLegacyStringEntries(t *testing.T) {
	var cache Cache
	if err := json.Unmarshal([]byte(`{"gin":"github.com/gin-gonic/gin"}`), &cache); err != nil {
		t.Fatal(err)
	}
	entry, ok := cache.GetEntry("gin")
	if !ok || entry.Module != "github.com/gin-gonic/gin" {
		t.Fatalf("unexpected cache entry: %+v", entry)
	}
}

func TestCacheReadsStructuredEntries(t *testing.T) {
	var cache Cache
	data := []byte(`{"air":{"name":"air","module":"github.com/air-verse/air","type":"command"}}`)
	if err := json.Unmarshal(data, &cache); err != nil {
		t.Fatal(err)
	}
	entry, ok := cache.GetEntry("air")
	if !ok || entry.Type != "command" {
		t.Fatalf("unexpected structured cache entry: %+v", entry)
	}
}
