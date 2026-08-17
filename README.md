# GoGet

Install Go **CLI tools** by name — not by full GitHub module path.

```bash
goget cli-name
# instead of: go install github.com/user/cli-name@latest

# Examples: standalone CLI tools
goget hugo            # Static site generator
goget k6              # Load testing tool
goget buf             # Protocol buffer compiler
goget golangci-lint   # Go linter aggregator
```

**Note:** GoGet installs **standalone CLI tools and utilities**. If you want to add a library dependency (like gin, gorm) to your project, use `go get` instead:

```bash
# Adding a dependency to your project
go get github.com/gin-gonic/gin

# Installing a CLI tool globally
goget myctl
```

GoGet searches GitHub for CLI repositories, ranks the results by popularity, shows an interactive menu when there's more than one match, detects typos, and remembers what you've installed. It has **zero external dependencies** — just the Go standard library — so there's nothing to fetch from a module proxy in order to build it.

## Install

### From Binary (Recommended)

Download pre-built binaries from the [Releases](https://github.com/kehindetemple/goget/releases) page:
- **macOS (Intel):** `goget-macos-amd64`
- **macOS (Apple Silicon):** `goget-macos-arm64`
- **Linux:** `goget-linux-amd64`
- **Windows:** `goget-windows-amd64.exe`

```bash
# macOS/Linux
chmod +x goget-macos-amd64
mv goget-macos-amd64 /usr/local/bin/goget

# Windows: Move goget-windows-amd64.exe to a directory in your PATH
```

### From Source

Requires Go 1.22+.

```bash
go install github.com/kehindetemple/goget@latest
```

Or clone and build locally:

```bash
git clone https://github.com/kehindetemple/goget.git
cd goget
go build -o goget .
# Then move `goget` or `goget.exe` to your PATH
```

## Commands

| Command | Description |
|---|---|
| `goget <tool-name>` | Search GitHub and install a CLI tool |
| `goget <tool-name> --save <profile>` | Install and save it into a profile |
| `goget info <tool-name>` | Show tool metadata without installing |
| `goget history` | Show recently installed tools |
| `goget login` | Save a GitHub token to raise API rate limits |
| `goget use <profile>` | Install every tool saved in a profile |
| `goget profile create <name>` | Create a new empty profile |
| `goget profile list` | List all profiles |
| `goget profile show <name>` | Show tools saved in a profile |
| `goget profile remove <profile> <tool>` | Remove one tool from a profile |
| `goget profile delete <name>` | Delete an entire profile (asks to confirm) |
| `goget profile export <name>` | Export profile to JSON file |
| `goget profile import <file>` | Import profile from JSON file |
| `goget profile share <name>` | Share profile via GitHub Gist |
| `goget profile download <gist-id>` | Download profile from GitHub Gist |
| `goget registry set <repo-url>` | Configure profile repository |
| `goget registry show` | Show current profile repository |
| `goget profile publish <name>` | Publish profile to registry repository |
| `goget profile fetch <username> <name>` | Fetch profile from someone's registry |

## How it works

For any `goget <tool-name>`, the resolution flow is:

1. **Local cache** — if you've resolved this exact name before, skip
   straight to installing.
2. **GitHub search** — search Go repositories whose name matches `<tool-name>`.
   - Exactly one exact match → install it directly.
   - Multiple matches → show an interactive menu ranked by **stars** and **recency** so you pick the right one.
3. **GitHub user/org search** — if nothing matched by name, check whether `<tool-name>`
   is a GitHub username/org, and if so list their top Go repositories to
   choose from.
4. **Typo detection** — if it's not a tool or user, run a broader search and rank results by edit distance (Levenshtein distance), suggesting "Did you mean…" options for typos.

When you pick a tool, GoGet:
- Caches the resolution (name → module path) for fast future installs
- Runs `go install github.com/user/tool@latest` to download and install it
- Logs it to your install history for reference

## Profiles

Profiles let you save and share collections of packages for different project types.

### Create & Manage Profiles Locally

```bash
# Create a new profile
goget profile create web-dev

# Add packages to profile
goget gin --save web-dev
goget gorm --save web-dev
goget viper --save web-dev

# View profile contents
goget profile show web-dev

# Install everything in profile
goget use web-dev

# Remove a package from profile
goget profile remove web-dev gin

# List all profiles
goget profile list

# Delete a profile
goget profile delete web-dev
```

### Import Profiles

#### Method 1: From GitHub Registry (Team/Organization Profiles)

Best for sharing profiles across your team or organization.

**Setup (one time):**
```bash
goget registry set github.com/yourname/goget-profiles
```

**Fetch a profile:**
```bash
goget profile fetch username profilename
```

**Example:**
```bash
goget registry set github.com/Ayokodes/goget-profiles
goget profile fetch Ayokodes myapi
goget use myapi
```

**Benefits:**
- Auto-sync with team profiles
- Centralized repository
- Easy to maintain and update

---

#### Method 2: From Email/File Share (Direct JSON)

Best for one-time sharing or when you receive a profile via email.

**Step 1 — Profile owner exports profile:**
```bash
goget profile export myapi
```
Creates: `goget-profile-myapi.json`

**Step 2 — Share the JSON file via email or file sharing service**

**Step 3 — Recipient imports the file:**
```bash
goget profile import goget-profile-myapi.json
```

**Step 4 — Install:**
```bash
goget use myapi
```

**Profile JSON Format:**
```json
{
  "name": "myapi",
  "packages": {
    "gin": {
      "name": "gin",
      "module": "github.com/gin-gonic/gin"
    },
    "gorm": {
      "name": "gorm",
      "module": "gorm.io/gorm"
    }
  }
}
```

---

#### Method 3: From GitHub Gist (Quick/Temporary Share)

Best for quick sharing, demos, or social media.

**Step 1 — Profile owner shares to Gist:**
```bash
goget profile share myapi
```

Output:
```
✅ Profile shared to Gist
Gist ID: a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6
Gist URL: https://gist.github.com/Ayokodes/a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6
```

**Step 2 — Share the Gist ID publicly**

**Step 3 — Others download using Gist ID:**
```bash
goget profile download a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6
```

**Step 4 — Install:**
```bash
goget use myapi
```

---

### Export Profiles

#### Export to JSON File

```bash
goget profile export myapi
```

Creates: `goget-profile-myapi.json`

**Use cases:**
- Email to coworkers
- Upload to shared drives (Google Drive, Dropbox)
- Commit to git repository
- Attach to GitHub issues

---

#### Export to GitHub Gist

```bash
goget profile share myapi
```

**Outputs:**
- Gist URL for sharing
- Gist ID for recipients to download

**Use cases:**
- Quick link sharing via Slack/Twitter
- GitHub discussions
- Documentation snippets
- Demo purposes

---

#### Export All Profiles

```bash
# macOS/Linux
for profile in $(goget profile list); do
  goget profile export $profile
done

# Windows PowerShell
$(goget profile list) | ForEach-Object { goget profile export $_ }
```

---

### Publish to Organization Repository

If you want to maintain a shared profile repository for your team:

**Step 1 — Set up your profiles repository:**
```bash
goget registry set github.com/yourname/goget-profiles
```

**Step 2 — Publish profile:**
```bash
goget profile publish myapi
```

This creates/updates `profiles/myapi.json` in your repository.

**Step 3 — Others can now fetch it:**
```bash
goget registry set github.com/yourname/goget-profiles
goget profile fetch yourname myapi
goget use myapi
```

**Profile Repository Structure:**
```
goget-profiles/
├── profiles/
│   ├── myapi.json
│   ├── web-dev.json
│   └── devops.json
├── README.md
└── .gitignore
```

---

## Import/Export Comparison

| Method | Setup Time | Sync | Best For | Link Format |
|--------|-----------|------|----------|------------|
| **GitHub Registry** | 1 min | Auto | Teams, permanent | `goget registry set ...` |
| **Email/File** | 2 min | Manual | One-time, quick | Send `.json` file |
| **GitHub Gist** | 30 sec | Manual | Quick demos | Gist ID: `a1b2c3d4...` |

---

## Local storage

Everything lives under `~/.config/goget/`:

```
~/.config/goget/
├── config.json       # GitHub token (from `goget login`)
├── cache.json        # name -> resolved module path
├── history.json      # recently installed packages
├── profiles.json     # named collections of saved packages
└── registry.json     # profile repository configuration
```

## GitHub rate limits

Unauthenticated requests to the GitHub API are limited to 60/hour, which
is easy to hit with repeated searches. Run `goget login` and paste a
[Personal Access Token](https://github.com/settings/tokens) (no scopes
needed for public repo/user search) to raise this to 5,000/hour.

## Project layout

```
goget/
├── main.go              CLI argument parsing and dispatch
├── resolve.go           shared "find the right repo" search flow
├── cmd_install.go       `goget <name>` / `goget <name> --save`
├── cmd_info.go          `goget info <name>`
├── cmd_profile.go       `goget profile ...`, `goget use`, `goget registry`
├── cmd_share.go         profile import/export/publish/fetch/share
├── cmd_misc.go          `goget history`, `goget login`
└── internal/
    ├── ghclient/        GitHub REST API client
    ├── storage/         JSON persistence (config/cache/history/profiles/registry)
    ├── fuzzy/           Levenshtein distance + ranking
    ├── installer/       wraps `go install`
    └── ui/              terminal prompts (menus, confirm, text input)
```
📱 Support
GoGet CLI Issues: https://github.com/kehindetemple/goget/issues
Profiles Issues: Open an issue in this repo
Questions: Discussions tab


⭐ Share & Star

Found this helpful?

⭐ Star this repo
🔄 Share profiles with your team
🎤 Spread the word about GoGet!

## Notes / known limitations

- `goget login` currently echoes the token as you type it (no external
  dependency is used for hidden input). Treat your terminal history and
  screen accordingly.
- Package resolution relies on the GitHub Search API, which occasionally
  returns repos that match on more than just the name (description/README
  matches can surface). The interactive menu always shows star counts and
  descriptions so you can sanity-check before installing.
- `goget install`'s underlying `go install <module>@latest` call depends
  on your local Go toolchain and network access to your configured
  `GOPROXY` (defaults to `proxy.golang.org`) — this is unrelated to GoGet
  itself.
- For profile registry features (`goget registry`, `goget profile publish`,
  `goget profile fetch`), users need write access to the repository if
  they want to publish profiles. Most users will only fetch/download profiles.