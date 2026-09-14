# Engineering Specification: soute (Project File Versioning CLI)

## 1. Overview & Core Philosophy

`soute` is a zero-friction, background CLI utility written in Go. It monitors structural project files (e.g., QLab databases, WATCHOUT show files, Ableton `.als` sets) and automatically creates content-addressed snapshots based on file modification timestamps and elapsed time.

**Design Imperatives:**

* **Strict Single-File Targeting:** `soute` strictly versions the core project/state file. It explicitly ignores directories and does not attempt to track bundled audio, video, or heavy media assets, preventing CPU and storage exhaustion.
* **Non-Disruptive:** Runs as an invisible background daemon. Expensive hashing operations are deferred until lightweight timestamp checks pass.
* **Platform-Native IPC:** Uses OS-level domain sockets and named pipes for zero-network communication across macOS and Windows.
* **Atomic & Deduplicated:** Handles OS-level atomic saves safely and deduplicates identical saves via SHA-256 content addressing with robust reference counting.

## 2. Technology Stack

* **Language:** Go (1.21+) with OS-specific build constraints (`//go:build darwin || linux`, `//go:build windows`).
* **CLI Router:** `spf13/cobra`.
* **CLI Output:** Go standard library `text/tabwriter` for clean, scriptable terminal tables (`list`, `status`).
* **Interactive TUI:** `charmbracelet/bubbletea` (Deferred to Phase 5, strictly for the `restore` interactive picker).
* **File Watcher:** `fsnotify/fsnotify`.
* **IPC Transport:** Standard `net` (Unix Domain Sockets) for macOS/Linux; `[github.com/Microsoft/go-winio](https://github.com/Microsoft/go-winio)` (Named Pipes) for Windows.
* **Process Management:** Native Go `os/exec` with OS-specific detachment flags.

## 3. Storage Architecture

To support watching multiple distinct files within the same parent directory without collision, `soute` utilizes a namespaced hidden folder pattern. All configuration, state, and file objects live in a hidden directory named `.soute_<target_filename>/` adjacent to the monitored target.

```text
.soute_MainShow.qlab5/
├── config.json         # User settings
├── daemon.json         # IPC state and endpoint tracking
├── manifest.json       # Index database mapping IDs to hashes
└── data/               # Content-addressed blob store
    ├── a1b2c3d4...     # SHA-256 hashed binary file object
    └── e5f6g7h8...

```

### 3.1. config.json Schema

| Field | Type | Description |
| --- | --- | --- |
| `target_path` | String | Path to the specific project file (e.g., `./MainShow.qlab5`). |
| `min_interval_seconds` | Int | Minimum time between snapshots; saves inside this window are skipped (`0` = snapshot on every content change). |
| `debounce_ms` | Int | Window to absorb rapid OS atomic save events (default: 300). |
| `max_snapshots` | Int | Limit before automated garbage collection triggers. |

### 3.2. manifest.json Schema

```json
{
  "version": "1.1",
  "snapshots": [
    {
      "id": "snap_1725800000",
      "timestamp": "2026-09-08T15:52:00Z",
      "content_hash": "a1b2c3d4e5f6...",
      "size_bytes": 452198,
      "mtime": "2026-09-08T15:51:58Z",
      "tag": "auto"
    }
  ]
}

```

## 4. Daemon & Process Management

The watcher runs in the background. Because platform-agnostic POSIX signals fall short, IPC is handled natively by the OS via Go build tags.

* **Forking & Detachment:** `soute watch` natively forks itself into the background.
* *macOS/Linux:* Sets `SysProcAttr{Setsid: true}`.
* *Windows:* Sets `CreationFlags: windows.CREATE_NEW_PROCESS_GROUP | windows.DETACHED_PROCESS`.


* **IPC Endpoints:** The daemon creates a listening socket/pipe.
* *macOS/Linux:* Unix Domain Socket at `.soute_<filename>/soute.sock`. Secured by directory file system permissions.
* *Windows:* Named Pipe at `\\.\pipe\soute-<Project-Path-Hash>`. Secured by Windows ACLs (SDDL string limiting to the current user).


* **State File (`daemon.json`):** Records the `pid`, `ipc_path`, and `status`. CLI commands read this file to route requests (Pause, Resume, Stop) to the correct local endpoint.

## 5. File Watcher & Evaluation Engine

### 5.1. Strict Single-File Targeting

`soute` enforces a strict 1:1 relationship between the daemon and a single target file. The `target_path` in `config.json` must point to a specific file (e.g., `show.qlab5`, `main.watch`, `set.als`), never a directory.

* `fsnotify` is attached exclusively to this single file, guaranteeing that bulk media rendering, audio mixdowns, or cache file generation within the surrounding project directory never wake the daemon or trigger evaluation logic.
* If a user attempts to run `soute init` on a directory, the CLI will explicitly fail and prompt them to select the primary project file within that directory.

### 5.2. Debounce & Lock Pipeline

Standard applications utilize atomic saves, firing rapid Create, Write, and Rename events. Media tools (especially SQLite-based bundles) also lock files aggressively.

1. `fsnotify` detects an event on the targeted file.
2. An internal timer resets to `config.debounce_ms` (300ms window). Any subsequent events reset this timer.
3. Upon expiration, if `os.Stat` or `os.Open` fails with a permission/lock error, the engine retries using an exponential backoff (100ms, 200ms, 400ms; max 3 retries) before dropping the event.

### 5.3. Evaluation Logic (mtime & Deferred Hashing)

To keep CPU overhead near zero, hashing is pushed to the very end of the pipeline.

1. **mtime Check:** Retrieve the OS file modification timestamp (`mtime`). If `mtime` matches the last snapshot's `mtime`, abort.
2. **Minimum Interval:** If the `mtime` has changed, check the time elapsed since the last snapshot. If at least `config.min_interval_seconds` has elapsed, proceed to hash; otherwise abort.
3. **Hash Deduplication:** Compute the SHA-256 hash of the target file. If `CurrentHash == LastSnapshotHash`, abort (the user saved without making material changes to the file contents).
4. **Commit:** Stream copy to `.soute_<filename>/data/<hash>.tmp`, perform an atomic `os.Rename` to `<hash>`, append the entry to `manifest.json`.

## 6. CLI Command Matrix

| Command | Action |
| --- | --- |
| `soute init <path>` | Creates the `.soute_<target_filename>/` tree in the target's directory and writes `config.json` with safe defaults (`debounce_ms: 300`, `min_interval_seconds: 300`, `max_snapshots: 50`). |
| `soute watch <path>` | Entrypoint for the watcher. Forks the OS-detached daemon, writes `daemon.json`, and spins up the UDS/Named Pipe listener. |
| `soute stop <path>` | Reads `daemon.json`, connects via UDS/Pipe, sends a stop command, waits for a clean exit, and removes the state file. |
| `soute status <path>` | Connects via IPC to check daemon health. Prints a simple text table showing Daemon Status, Uptime, Target File, and the 3 most recent snapshots. |
| `soute list <path>` | Prints the full `manifest.json` history as a clean, scriptable `text/tabwriter` table (ID, Timestamp, File Size). |
| `soute restore <path> [id]` | **Execution Pipeline:** Pre-flight backup (hashes current file, copies to data store if it differs from the manifest). Sends Pause via IPC. Atomic swap of the target with the selected snapshot. Sends Resume via IPC.<br>

<br>

<br>**Hybrid Input:** If `[id]` is provided, executes immediately. If omitted, triggers the interactive TUI picker (implemented in Phase 5). |
| `soute prune <path>` | Triggers garbage collection. **Enforces Reference Counting:** Removes the oldest manifest entries exceeding `max_snapshots`. Deletes a blob in `data/` *only* if zero manifest entries point to its hash. |

*Note: With namespaced tracking, all commands except `init` now require the `<path>` argument (or execution from a directory with only one `.soute_*` folder where it can auto-detect the target) to locate the correct state directory.*

## 7. Implementation Milestones

* **Phase 1: Foundation.** Implement `soute init <path>`, namespaced `.soute_<filename>/` directory creation, default config generation, single-file targeting enforcement, and manifest data structures.
* **Phase 2: Storage Engine.** Implement deferred SHA-256 hashing, atomic temp-file writes to the object store, and reference-counted garbage collection (`soute prune`).
* **Phase 3: Cross-Platform Daemon.** Implement `fsnotify`, the 300ms debounce/lock retry pipeline, process detachment flags, and the UDS/Named Pipe IPC architecture (`soute watch`, `soute stop`).
* **Phase 4: Safety & Interface.** Implement the core logic of `soute restore` (pre-flight backup, IPC pausing, file swapping), and scriptable `text/tabwriter` outputs for `soute list` and `soute status`.
* **Phase 5: Polish.** Integrate `charmbracelet/bubbletea` to build the interactive terminal table picker for `soute restore` when a user runs the command without supplying a specific snapshot ID.
* **Phase 6: Release Infra.** Configure Continuous Integration (e.g., GitHub Actions) via `goreleaser` to automate cross-platform compilation (`GOOS=darwin`, `GOOS=windows`), package releases, and generate native installers (Homebrew tap for macOS, Scoop for Windows).
