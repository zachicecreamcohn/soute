# soute

Zero-friction versioning for large binary & media files (QLab shows, WATCHOUT
files, Ableton sets, Photoshop documents). `soute` runs an invisible background
daemon that creates content-addressed snapshots whenever a watched file changes
by more than a configurable size or percentage delta.

## Build

Requires Go 1.23+ (the charmbracelet and fsnotify dependencies require it).

```sh
go build -o soute .
```

## Usage

```text
soute init                        # interactive setup wizard
soute watch                       # fork a background watcher (--daemon=false for foreground)
soute status                      # daemon health + last 3 snapshots
soute list                        # full history
soute restore [snapshot_id]       # pause daemon, back up current state, restore
soute export [id] [destination]   # copy a snapshot out (daemon untouched)
soute prune [--keep N] [--all]    # garbage-collect
soute config [view|edit]          # show or edit settings
soute stop                        # stop the daemon
```

Every command accepts `--dir <path>` (default `.`) pointing at the directory
that contains `.snapshots`. `init` is non-interactive when given `--target`
(plus optional `--min-bytes`, `--min-pct`, `--cooldown`, `--max-snapshots`).

## Storage

Each monitored file has a `.snapshots/` directory beside it:

```text
.snapshots/
├── config.json      # settings (thresholds, cooldown, retention)
├── daemon.pid       # single-instance lock
├── manifest.json    # history index (source of truth)
└── data/            # content-addressed blob store (SHA-256)
```

Snapshots are deduplicated by content: saving identical bytes never duplicates
storage. Blobs are written to a `.tmp` file and atomically renamed, so a crash
can never leave a partial object behind.

## How the daemon works

- Watches the target's **directory**, since atomic saves rename files in and out.
- Coalesces rapid events through a fixed **300 ms** debounce window.
- Retries transient lock/stat failures at **100/200/400 ms**.
- Captures when `|Δsize| >= min_delta_bytes` **or** `Δ% >= min_delta_pct`,
  then aborts if the SHA-256 matches the previous snapshot.
- Enforces `cooldown_seconds` as the minimum interval between snapshots.

## Design notes

**Cross-platform IPC.** The spec's daemon control (`SIGTERM`/`SIGUSR1`/`SIGUSR2`,
`Setsid`) is POSIX-only, so it is abstracted behind build tags:

- **macOS / Linux**: POSIX signals and `Setsid`, exactly as specified.
- **Windows**: a per-project named pipe (`soute-<hash>`) carries the same
  stop/pause/resume commands, and the child detaches via `DETACHED_PROCESS`.

**Restore tagging.** Snapshots are tagged `auto`, `manual`, or
`pre_restore_backup`. A `restore` entry is additionally appended after every
restore so the timeline records when a restore happened and to what.
