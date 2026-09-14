# soute

Zero-friction project file versioning.

`soute` is a background CLI that watches a **single structural project file**
(QLab `.qlab5` databases, WATCHOUT show files, Ableton `.als` sets) and
automatically creates content-addressed, deduplicated snapshots. It deliberately
ignores directories and bundled media so it never burns CPU or disk tracking
heavy assets.

## Install

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

> **Note:** `soute watch` re-executes itself in the background, so it requires a
> real binary (built or installed), not `go run`. `go run` re-execs a temporary
> binary that is deleted immediately.

## Quick start

```sh
soute init show.qlab5      # create the .soute_show.qlab5/ namespace
soute watch show.qlab5     # fork the detached daemon
soute status               # daemon health + 3 most recent snapshots
soute list                 # full snapshot history (ID, timestamp, size)
soute restore show.qlab5   # interactive picker (or pass an ID to skip it)
soute stop                 # clean shutdown
soute prune                # reference-counted garbage collection
```

Each tracked file gets a namespaced `.soute_<filename>/` directory beside it:

```text
.soute_show.qlab5/
├── config.json     # target path, debounce, time threshold, snapshot limit
├── daemon.json     # daemon pid + IPC endpoint (while running)
├── manifest.json   # index of snapshots (id → content hash)
└── data/           # content-addressed blob store (SHA-256)
```

## How it works

- **Strict single-file targeting** — the watcher ignores every sibling/media
  event, so rendering assets never wake evaluation.
- **Deferred hashing** — an mtime check and an elapsed-time threshold gate the
  expensive SHA-256 pass; identical content deduplicates against the last
  snapshot.
- **Atomic & deduplicated** — snapshots are written via temp-file + `fsync` +
  rename; identical saves share one blob.
- **Reference-counted GC** — `prune` deletes a blob only when no surviving
  snapshot references it.
- **Native IPC** — Unix domain sockets (macOS/Linux) and named pipes (Windows),
  with zero network.

## Configuration

`config.json` defaults: `debounce_ms: 300`, `max_time_seconds: 300`,
`max_snapshots: 50`. Set `max_time_seconds: 0` to snapshot on every content
change.
