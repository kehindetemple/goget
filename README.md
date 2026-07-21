# GoGet

Install Go packages by name — not by full GitHub module path.

```bash
goget gin
# instead of: go install github.com/gin-gonic/gin@latest
```

GoGet searches GitHub, ranks the results, asks you to pick when there's
more than one reasonable match, detects typos, and remembers what you've
installed. It has **zero external dependencies** — just the Go standard
library — so there's nothing to fetch from a module proxy in order to
build it.

## Build

Requires Go 1.22+.

```bash
go build -o goget .          # macOS/Linux
go build -o goget.exe .      # Windows
```

Optionally, install it onto your PATH:

```bash
go install .
```

## Commands

| Command | Description |
|---|---|
| `goget <name>` | Search GitHub and install a package |
| `goget <name> --save <profile>` | Install and save it into a profile |
| `goget info <name>` | Show package metadata without installing |
| `goget history` | Show recently installed packages |
| `goget login` | Save a GitHub token to raise API rate limits |
| `goget use <profile>` | Install every package saved in a profile |
| `goget profile create <name>` | Create a new empty profile |
| `goget profile list` | List all profiles |
| `goget profile show <name>` | Show packages saved in a profile |
| `goget profile remove <profile> <pkg>` | Remove one package from a profile |
| `goget profile delete <name>` | Delete an entire profile (asks to confirm) |

## How resolution works

For any `goget <name>`, the tool works through this sequence until it
finds an install target:

1. **Local cache** — if you've resolved this exact name before, skip
   straight to installing (feature: Local Cache).
2. **GitHub search** — search Go repositories whose name matches `<name>`.
   - Exactly one exact match → install it directly.
   - Several matches (exact or close) → show a numbered menu to pick from
     (feature: Interactive Package Selection).
3. **Owner search** — if nothing matched by name, check whether `<name>`
   is a GitHub username/org, and if so list their top Go repositories to
   choose from (feature: GitHub Owner Search).
4. **Typo suggestions** — if it's not a package or a user, run a broader
   search and rank results by edit distance to `<name>`, presenting a
   "Did you mean…" list (feature: Intelligent Typo Detection).

Whatever you pick gets cached, installed with `go install <module>@latest`,
and logged to your install history.

## Local storage

Everything lives under `~/.config/goget/`:

```
~/.config/goget/
├── config.json     # GitHub token (from `goget login`)
├── cache.json       # name -> resolved module path
├── history.json    # recently installed packages
└── profiles.json   # named collections of saved packages
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
├── resolve.go            shared "find the right repo" search flow
├── cmd_install.go        `goget <name>` / `goget <name> --save`
├── cmd_info.go           `goget info <name>`
├── cmd_profile.go        `goget profile ...` and `goget use`
├── cmd_misc.go           `goget history`, `goget login`
└── internal/
    ├── ghclient/          GitHub REST API client
    ├── storage/           JSON persistence (config/cache/history/profiles)
    ├── fuzzy/              Levenshtein distance + ranking
    ├── installer/          wraps `go install`
    └── ui/                 terminal prompts (menus, confirm, text input)
```

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
