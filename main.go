// Command goget is an intelligent Go CLI tool that installs Go packages
// without requiring the user to remember full GitHub module paths.
//
// See the accompanying PRD (GoGet V1) for full feature details.
package main

import (
	"fmt"
	"os"
)

const usage = `GoGet — install Go packages by name, not by full module path.

Usage:
  goget <name>                        Search GitHub and install a package
  goget <name> --save <profile>       Install and save it into a profile
  goget info <name>                   Show package metadata without installing
  goget history                       Show recently installed packages
  goget login                         Save a GitHub token to raise API rate limits

  goget use <profile>                 Install every package saved in a profile

  goget profile create <name>         Create a new empty profile
  goget profile list                  List all profiles
  goget profile show <name>           Show packages saved in a profile
  goget profile remove <p> <pkg>      Remove one package from a profile
  goget profile delete <name>         Delete an entire profile

  goget profile share <name>          Upload profile to GitHub Gist (shareable link)
  goget profile download <gistID>     Download a shared profile from GitHub Gist
  goget profile export <name>         Export profile to a local JSON file
  goget profile import <file>         Import profile from a local JSON file

  goget profile publish <name>        Publish profile to your GitHub repo
  goget profile fetch <user> <name>   Fetch profile from another user's GitHub repo
  goget registry set <repo-url>       Configure your profile repository (e.g., github.com/user/goget-profiles)

Examples:
  goget gin
  goget joho
  goget godotvn
  goget gin --save api
  goget use api
  goget profile share api
  goget profile download abc123def456
  goget registry set github.com/yourname/goget-profiles
  goget profile publish api
  goget profile fetch othername api
`

func main() {
	args := os.Args[1:]
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
		fmt.Println("GoGet v1.0.0")
		fmt.Println("Intelligent Go package installer")
		fmt.Println("https://github.com/kehindetemple/goget")
		return

	case "info":
		if len(args) < 2 {
			fmt.Println("usage: goget info <name>")
			os.Exit(1)
		}
		err = cmdInfo(args[1])

	case "login":
		err = cmdLogin()

	case "history":
		err = cmdHistory()

	case "use":
		if len(args) < 2 {
			fmt.Println("usage: goget use <profile>")
			os.Exit(1)
		}
		err = cmdUse(args[1])

	case "registry":
		err = dispatchRegistry(args[1:])

	case "profile":
		err = dispatchProfile(args[1:])

	default:
		pkg, saveProfile, perr := parseInstallArgs(args)
		if perr != nil {
			fmt.Println("error:", perr)
			os.Exit(1)
		}
		err = cmdInstall(pkg, saveProfile)
	}

	if err != nil {
		fmt.Println("error:", err)
		os.Exit(1)
	}
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