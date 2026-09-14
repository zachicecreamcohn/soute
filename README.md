# soute

`soute` is a small background CLI that versions a single project file. Point it
at something like a QLab `.qlab5` file and it saves a content-addressed, deduplicated snapshot on every change.

## Why

As a software engineer, I love git for versioning. But for my theatre work, I work exclusively with binary files (e.g. `.watchout`, `.qlab5`, `.millumin`) which have varying levels of versioning.

I built this tool as a standard versioning system for working with these files. It's cross-platform, simple to use, and hopefully helpful to others!


### The name?
"Soute" in French means a cargo hold. When we do a snapshot of a project file, we're adding it to our "hold" of backups! Also it sounds cool.

## Installation

```sh
# From source
go install github.com/zachicecreamcohn/soute/cmd/soute@latest

# macOS (Homebrew)
brew tap zachicecreamcohn/homebrew-tap
brew trust zachicecreamcohn/tap
brew install soute

# Windows (Scoop)
scoop bucket add zachicecreamcohn https://github.com/zachicecreamcohn/scoop-bucket
scoop install soute
```

> `soute watch` re-executes itself in the background, so it needs a real binary —
> install it first, not `go run`.

## Usage

### Initialize

Point soute at your project file:

```sh
soute init show.qlab5
```

This creates a `.soute_show.qlab5/` directory next to it:

```text
.soute_show.qlab5/
├── config.json     # your settings
├── daemon.json     # daemon state (only while running)
├── manifest.json   # snapshot index
└── data/           # content-addressed snapshot blobs (SHA-256)
```

`init` writes `config.json` with safe defaults:

```json
{
  "target_path": "/path/to/show.qlab5",
  "min_interval_seconds": 300,
  "debounce_ms": 300,
  "max_snapshots": 50
}
```

`min_interval_seconds` is the minimum gap between snapshots (set it to `0` to
snapshot on every change); `debounce_ms` absorbs rapid atomic-save events; and
`max_snapshots` caps how many snapshots are kept (if `50`, we keep only the `50` most recent backups).

### Watch
Start watching the file for saves

```sh
soute watch show.qlab5
```

### Manual Saves

You don't have to wait for a change. Save the current state now and tag it:

```sh
soute backup show.qlab5 -t "Tech day 1 EOD"
```

Tags show up in the restore picker.

### Restore
To restore the file to a backed up point:

```sh
soute restore show.qlab5
```

This opens a TUI to pick from backups. When restoring, we automatically save a pre-restore snapshot just in case.
