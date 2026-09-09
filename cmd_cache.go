package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/kehindetemple/goget/internal/registry"
	"github.com/kehindetemple/goget/internal/storage"
)

func dispatchCache(args []string) error {
	if len(args) == 0 {
		return cmdCacheShow()
	}
	if len(args) == 1 && args[0] == "clear" {
		return storage.ClearCache()
	}
	return fmt.Errorf("usage: goget cache [clear]")
}

func cmdCacheShow() error {
	cache, err := storage.LoadCache()
	if err != nil {
		return err
	}
	commands, libraries := 0, 0
	catalog := registry.Load()
	var last time.Time
	for _, entry := range cache {
		entryType := entry.Type
		if entryType == "" {
			if known, ok := catalog.Resolve(entry.Module); ok {
				entryType = string(known.Type)
			} else {
				entryType = string(packageFromModule(entry.Module).Type)
			}
		}
		if strings.EqualFold(entryType, string(registry.Command)) {
			commands++
		} else {
			libraries++
		}
		if entry.UpdatedAt.After(last) {
			last = entry.UpdatedAt
		}
	}
	fmt.Println("GoGet Cache")
	fmt.Println()
	fmt.Printf("Packages: %d\n", libraries)
	fmt.Printf("Commands: %d\n", commands)
	if last.IsZero() {
		fmt.Println("Last updated: never")
	} else {
		fmt.Printf("Last updated: %s\n", last.Local().Format("2006-01-02 15:04"))
	}
	if len(cache) == 0 {
		return nil
	}
	fmt.Println()
	for key, entry := range cache {
		entryType := entry.Type
		if entryType == "" {
			if known, ok := catalog.Resolve(entry.Module); ok {
				entryType = string(known.Type)
			} else {
				entryType = string(packageFromModule(entry.Module).Type)
			}
		}
		fmt.Printf("%-20s -> %s (%s)\n", key, entry.Module, entryType)
	}
	return nil
}
