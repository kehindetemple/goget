// Command goget is an intelligent Go package discovery and installation tool.
package main

import (
	"fmt"
	"os"
	"strings"
)

const usage = `GoGet - discover and install Go packages by name.

Usage:
  goget <name>[@version]             Resolve and install a package or command
  goget --offline <name>             Resolve from local data without discovery
  goget <name> --save <profile>      Install and save it into a profile
  goget search <query>               Search the local GoGet registry
  goget info <name>                  Show package metadata without installing
  goget update <name>                Update a package or command to latest
  goget outdated                     Show available project module updates
  goget doctor                       Diagnose GoGet and Go environment
  goget cache                        Show local resolution cache
  goget cache clear                  Clear local resolution cache
  goget init                         Initialize a Go module interactively
  goget remove <name>                Remove a project dependency safely
  goget history                      Show recently installed packages
  goget login                        Save a GitHub token for fallback discovery

  goget use <profile>                Install every tool saved in a profile
  goget profile <subcommand>         Create, list, share, import, or publish profiles
  goget registry <subcommand>        Configure the profile repository

Examples:
  goget gin
  goget air
  goget github.com/gin-gonic/gin
  goget gin@v1.10.0
  goget search database
  goget --offline gin
`

func main() {
	args, offline := parseGlobalArgs(os.Args[1:])
	if len(args) == 0 {
		fmt.Print(usage)
		os.Exit(1)
	}

	var err error
	switch args[0] {
	case "help", "-h", "--help":
		fmt.Print(usage)
		return
	case "version", "-v", "--version":
		fmt.Println("GoGet v3.0.0")
		fmt.Println("Intelligent Go package discovery and installation")
		fmt.Println("https://github.com/kehindetemple/goget")
		return
	case "info":
		if len(args) < 2 {
			err = fmt.Errorf("usage: goget info <name>")
			break
		}
		err = cmdInfo(args[1])
	case "search":
		if len(args) < 2 {
			err = fmt.Errorf("usage: goget search <query>")
			break
		}
		err = cmdSearchWithOptions(strings.Join(args[1:], " "), offline)
	case "doctor":
		err = cmdDoctor()
	case "cache":
		err = dispatchCache(args[1:])
	case "update":
		if len(args) < 2 {
			err = fmt.Errorf("usage: goget update <name>")
			break
		}
		err = cmdUpdate(args[1])
	case "outdated":
		err = cmdOutdated()
	case "remove":
		if len(args) < 2 {
			err = fmt.Errorf("usage: goget remove <name>")
			break
		}
		err = cmdRemove(args[1])
	case "init":
		err = cmdInit()
	case "login":
		err = cmdLogin()
	case "history":
		err = cmdHistory()
	case "use":
		if len(args) < 2 {
			err = fmt.Errorf("usage: goget use <profile>")
			break
		}
		err = cmdUse(args[1])
	case "registry":
		err = dispatchRegistry(args[1:])
	case "profile":
		err = dispatchProfile(args[1:])
	default:
		pkg, saveProfile, perr := parseInstallArgs(args)
		if perr != nil {
			err = perr
			break
		}
		err = cmdInstallWithOptions(pkg, saveProfile, offline)
	}

	if err != nil {
		fmt.Println("error:", err)
		os.Exit(1)
	}
}

func parseGlobalArgs(args []string) ([]string, bool) {
	offline := false
	filtered := make([]string, 0, len(args))
	for _, arg := range args {
		if arg == "--offline" {
			offline = true
			continue
		}
		filtered = append(filtered, arg)
	}
	return filtered, offline
}

func dispatchProfile(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: goget profile <create|list|show|delete|remove> ...")
	}
	switch args[0] {
	case "create":
		if len(args) < 2 {
			return fmt.Errorf("usage: goget profile create <name>")
		}
		return cmdProfileCreate(args[1])
	case "list":
		return cmdProfileList()
	case "show":
		if len(args) < 2 {
			return fmt.Errorf("usage: goget profile show <name>")
		}
		return cmdProfileShow(args[1])
	case "delete":
		if len(args) < 2 {
			return fmt.Errorf("usage: goget profile delete <name>")
		}
		return cmdProfileDelete(args[1])
	case "remove":
		if len(args) < 3 {
			return fmt.Errorf("usage: goget profile remove <profile> <package>")
		}
		return cmdProfileRemove(args[1], args[2])
	case "share":
		if len(args) < 2 {
			return fmt.Errorf("usage: goget profile share <name>")
		}
		return cmdProfileShare(args[1])
	case "download":
		if len(args) < 2 {
			return fmt.Errorf("usage: goget profile download <gistID>")
		}
		return cmdProfileDownload(args[1])
	case "export":
		if len(args) < 2 {
			return fmt.Errorf("usage: goget profile export <name>")
		}
		return cmdProfileExport(args[1])
	case "import":
		if len(args) < 2 {
			return fmt.Errorf("usage: goget profile import <file>")
		}
		return cmdProfileImport(args[1])
	case "publish":
		if len(args) < 2 {
			return fmt.Errorf("usage: goget profile publish <name>")
		}
		return cmdProfilePublish(args[1])
	case "fetch":
		if len(args) < 3 {
			return fmt.Errorf("usage: goget profile fetch <username> <profilename>")
		}
		return cmdProfileFetch(args[1], args[2])
	default:
		return fmt.Errorf("unknown profile subcommand %q", args[0])
	}
}
