# soute

Zero-friction versioning for large binary & media files (QLab, WATCHOUT,
Ableton, Photoshop). A background daemon snapshots a watched file whenever a
save changes it beyond a configurable size or percentage delta.

## Install

Prebuilt binaries for macOS, Linux, and Windows (amd64 and arm64) are on the
[releases page](https://github.com/zachicecreamcohn/soute/releases/latest).

Or with Go:

```sh
go install github.com/zachicecreamcohn/soute@latest
```

## Quick start

```sh
soute init <path-to-file>
soute watch                 # start the background daemon
# …edit and save the file…
soute list                  # view snapshots
soute restore               # roll back to a snapshot
```

## Commands

- `init <target>` — create a `.snapshots` config for a file
- `watch` — start the background watcher (`--daemon=false` for foreground)
- `status` — daemon health + recent snapshots
- `list` — full snapshot history
- `restore [id]` — restore a snapshot (backs up the current file first)
- `export [id] [destination]` — copy a snapshot out, leaving the working file alone
- `prune [--keep N] [--all]` — garbage-collect snapshots
- `config [view|edit]` — view or edit settings
- `stop` — stop the daemon

All commands accept `--dir <path>` (default `.`) for the directory containing
`.snapshots`.

## How it works

Each watched file gets a `.snapshots/` directory beside it holding `config.json`
(settings), `manifest.json` (history), and a content-addressed `data/` blob
store. Identical saves deduplicate by SHA-256 and blobs are written atomically.
The daemon watches the file's directory, coalesces rapid save events through a
300 ms debounce, and snapshots when the change crosses `min_delta_bytes` or
`min_delta_pct`.
