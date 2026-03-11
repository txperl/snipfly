# SnipFly PRD

## 1. Document Purpose

This document is based on the current Go implementation in the repository, describing SnipFly's shipped product capabilities, interaction behavior, and technical architecture.

This is not an "ideal design draft" but rather an "implementation-aligned PRD." All behaviors described in this document are grounded in the current state of `main.go`, `internal/snippet`, `internal/runner`, `internal/tui`, and the release configuration.

---

## 2. Product Overview

SnipFly is a lightweight TUI tool that scans local directories for code snippets and provides the following operations within a single terminal interface:

- Browse snippet list
- Start, stop, and restart snippets
- View runtime output
- Run full-screen interactive tools

Supported snippet types include shell, TS/JS, Python, Go, Ruby, Lua, and extensionless scripts with a shebang line.

---

## 3. Current Version Scope

### 3.1 Implemented Features

- Scan snippets from the current directory, a specified directory, or the user's home directory
- Automatically detect `.snipfly/` subdirectory under the target directory
- Support root-level files and first-level subdirectory grouping
- Parse snippet header annotation metadata
- Automatically infer interpreters, or allow overrides via annotations
- Run `oneshot`, `service`, and `interactive` snippet types
- View status, output, and metadata in the TUI
- Exit confirmation when running tasks exist
- Publish binaries and Homebrew formula via GoReleaser

### 3.2 Not Yet Implemented

- Snippet filtering or search
- `j/k` navigation
- Auto-restart crashed services
- Multi-level directory recursive grouping
- Stop/restart management for interactive snippets

---

## 4. User-Facing Behavior

### 4.1 CLI Usage

```bash
snipfly
snipfly .
snipfly ./examples
snipfly --exact ./examples
snipfly --global
snipfly --version
snipfly --help
```

Currently implemented parameters:

| Parameter   | Short | Description                                                           |
| ----------- | ----- | --------------------------------------------------------------------- |
| `--exact`   | `-e`  | Disable `.snipfly/` subdirectory auto-detection; scan target directly |
| `--global`  | `-g`  | Scan the user's home directory `~`                                    |
| `--version` | `-v`  | Print version information and exit                                    |
| `--help`    | `-h`  | Help output provided by `pflag`                                       |

Scan directory resolution rules:

1. When `--global` is used, the target directory is the user's home directory.
2. Otherwise, if a positional argument is provided, use that directory.
3. Otherwise, default to scanning the current directory `.`.
4. If `--exact` is not used and a `.snipfly/` subdirectory exists under the target, the actual scan directory becomes that subdirectory.

When the scan result is empty, the program exits immediately with the message `No snippets found in ...`.

### 4.2 Snippet Discovery Rules

SnipFly currently scans only a two-level structure:

- Root-level files in the target directory
- Files within first-level subdirectories of the target directory

Behavioral details:

- Root-level files have `group=""`
- First-level subdirectory names become group names
- Deeper directories are not recursively scanned
- Hidden files and hidden directories are skipped
- Only files with "known extensions" or extensionless files with a shebang are treated as snippets

Built-in extension mappings:

| Extension | Default Interpreter |
| --------- | ------------------- |
| `.sh`     | `bash`              |
| `.ts`     | `npx tsx`           |
| `.js`     | `node`              |
| `.py`     | `python3`           |
| `.go`     | `go run`            |
| `.rb`     | `ruby`              |
| `.lua`    | `lua`               |

Sorting rules:

- Root-level snippets come before grouped snippets
- Groups are sorted by name in ascending order
- Snippets within a group are sorted by `Name` in ascending order

### 4.3 Annotation Format

SnipFly reads annotations from consecutive comment lines at the top of a file, stopping at the first line that is non-empty and non-comment.

- Supports comment prefix `#`
- Supports comment prefix `//`
- Skips shebang lines
- `@env` can appear multiple times
- Parsing splits only on the first `:`, preserving extra colons in the value

Example:

```bash
# @name: Clock Server
# @desc: Print current time every second
# @type: service
# @dir: /tmp
# @env: TZ=UTC
# @env: LANG=en_US.UTF-8
# @pty: true

while true; do date; sleep 1; done
```

Minimal example:

```bash
echo "Hello from SnipFly!"
```

Supported fields:

| Annotation     | Purpose                                 | Default                          |
| -------------- | --------------------------------------- | -------------------------------- |
| `@name`        | Display name in the list                | Filename (without extension)     |
| `@desc`        | Brief description                       | Empty                            |
| `@type`        | `oneshot` / `service` / `interactive`   | `oneshot`                        |
| `@dir`         | Working directory, supports `~`         | Directory containing the snippet |
| `@env`         | Append environment variable `KEY=VALUE` | Empty                            |
| `@interpreter` | Explicitly specify interpreter command  | Auto-inferred                    |
| `@pty`         | Whether to run via PTY                  | `false`                          |

Additional notes:

- The current implementation does not validate whether `@type` matches a predefined enum, but the TUI only handles `interactive` as a special branch
- `@pty` only takes effect when the value equals `true` (case-insensitive)

### 4.4 Interpreter Resolution Priority

Current priority order:

1. `@interpreter`
2. File's first-line shebang
3. Extension mapping

Resolution details:

- `@interpreter: npx tsx` is split into command `npx` and argument `tsx`
- `#!/usr/bin/env bash` resolves to `bash`
- `#!/bin/bash` resolves to `bash`
- Additional arguments after the shebang are preserved

If the interpreter cannot be resolved:

- The snippet still appears in the list
- Its `Snippet.Error` field is populated
- The left-side list shows a red failure icon
- The snippet cannot be started; the right-side output area displays the error message

### 4.5 Execution Model and States

SnipFly currently supports three snippet types:

| Type          | Behavior                                                |
| ------------- | ------------------------------------------------------- |
| `oneshot`     | Starts and waits for natural exit                       |
| `service`     | Treated as a long-running task; can be stopped manually |
| `interactive` | Takes over the entire terminal via `tea.ExecProcess`    |

Current state machine:

- `oneshot`: `Idle -> Running -> Done/Failed`
- `service`: `Idle -> Running -> Stopped/Exited/Crashed`
- `interactive`: Does not enter the Runner state machine; status bar shows `◇ Interactive`

Terminal state rules:

- `oneshot` with exit code `0`: `Done`
- `oneshot` with non-zero exit: `Failed`
- `service` with exit code `0`: `Exited`
- `service` with non-zero exit: `Crashed`
- User-initiated stop: `Stopped`

### 4.6 TUI Structure and Interaction

The interface uses a left-right split layout:

- Left side: snippet list
- Right side: metadata + output area
- Bottom: status bar

ASCII diagram:

```text
┌────────────────────────┬───────────────────────────────────────────────────────┐
│ ◆ SnipFly              │ ◆ Output                                              │
│  ── demo ──            │ @name: Clock Server                                    │
│  ● Clock Server        │ @desc: Print current time every second                 │
│    minimum             │ @type: service                                         │
│  ── test ──            │ @dir: /tmp                                             │
│  ✓ fast-exit           │ @env: TZ=UTC                                           │
│  ■ slow-task           │ @env: LANG=en_US.UTF-8                                 │
│    hello               │ @interpreter: bash                                     │
│    crash-test          │ @pty: true                                             │
│                        │ ───────────────────────────────────────────────────── │
│                        │ Thu Mar 12 01:23:45 UTC 2026                           │
├────────────────────────┴───────────────────────────────────────────────────────┤
│ ↑/↓:navigate  Tab:switch  Space:run/stop  r:re-run  q:quit    ● Running PID:1234 │
└────────────────────────────────────────────────────────────────────────────────┘
```

Legend:

- The left side shows group headers and snippet status icons
- The right side displays metadata at the top, followed by output content
- The bottom status bar shows keyboard shortcuts on the left and the selected snippet's status on the right
- When focus switches to the output area, the title changes to `◆ Output`, and the list title reverts to an unfocused style

Layout rules:

- Left side width is 30% of total width
- Left side minimum: 20 columns, maximum: 40 columns
- Right side takes the remaining width
- Bottom status bar is fixed at 1 row

Focus rules:

- On launch, focus defaults to the left-side list
- `Tab` toggles focus between the list area and the output area
- The focused panel title displays as `◆ SnipFly` or `◆ Output`

Current list area keybindings:

| Key       | Action                                                         |
| --------- | -------------------------------------------------------------- |
| `↑`       | Move selection up                                              |
| `↓`       | Move selection down                                            |
| `Space`   | Start snippet; if currently running (non-interactive), stop it |
| `r` / `R` | Restart current non-interactive snippet                        |
| `Tab`     | Switch to output area                                          |
| `q` / `Q` | Quit; shows confirmation dialog if tasks are running           |

Output area behavior:

- Uses `bubbles/viewport` to render content
- Output scrolling keys follow the current status bar text, i.e., `↑/↓`
- When the viewport is at the bottom, new output auto-scrolls to the bottom
- If the user manually scrolls up, it does not force-jump back to the bottom

Exit confirmation behavior:

- Only appears when running Runner processes exist
- Message: `Running processes detected! Quit and stop all? (y/n)`
- `y` / `Y`: Stop all running processes and exit
- `n` / `N` / `Esc`: Close the confirmation dialog and return to the main interface

### 4.7 Metadata Panel Display Rules

The right-side output area shows metadata for the currently selected snippet at the top, with a separator line below.

Display order is fixed:

1. `@name`
2. `@desc`
3. `@type`
4. `@dir`
5. `@env`
6. `@interpreter`
7. `@pty`

Unlike earlier designs, the current implementation displays "runtime-resolved results" rather than just "explicitly declared annotations." Specific rules:

- `@name` is always displayed
- `@desc` is displayed only when non-empty
- `@type` is displayed only when it is not the default value `oneshot`
- `@dir` is always displayed as long as the snippet exists, since it always resolves to a valid directory
- `@env` entries are displayed in declaration order
- `@interpreter` is displayed whenever successfully resolved
- `@pty` is displayed only when `true`

### 4.8 Status Display

The left-side list and bottom status bar use the same status semantics:

| State     | Icon  | Description                |
| --------- | ----- | -------------------------- |
| `Running` | `●`   | Currently running          |
| `Crashed` | `✗`   | Service exited abnormally  |
| `Failed`  | `✗`   | Oneshot non-zero exit      |
| `Done`    | `✓`   | Oneshot completed normally |
| `Exited`  | `✓`   | Service exited normally    |
| `Stopped` | `■`   | User manually stopped      |
| `Idle`    | Space | Not running                |

Additional notes:

- Snippets with interpreter resolution failures also show `✗`, but this is a build-time error indicator, not a Runner state
- The status bar shows PID when `Running`
- The status bar shows the exit code when `Crashed` / `Failed`

### 4.9 Output Rendering Rules

In the current implementation, output area content comes from the following sources:

- Normal snippets: from the Runner's ring buffer
- Interactive snippets: show "launch prompt" or "most recent exit result"
- No output but process exists: show current state and `(no output yet)`
- Not yet started: show `State: Idle` and a launch prompt

Terminal state appended text:

- `Done` / `Exited`: `--- Process exited (code: 0) ---`
- `Failed` / `Crashed`: show actual exit code
- `Stopped`: `--- Process stopped ---`
- `Crashed` additionally shows `Press Space to re-run.`

---

## 5. Edge Cases and Error Handling

### 5.1 Service Immediate Exit

If a `service` exits quickly:

- Exit code `0` is marked as `Exited`
- Non-zero exit is marked as `Crashed`
- The output area retains its output and exit information
- The user can press `Space` to restart

### 5.2 Long-Running Oneshot

A `oneshot` is not forcibly converted to `service` due to long runtime.

Current behavior:

- Still runs as `oneshot`
- Output continues to refresh
- The user can press `Space` to manually stop
- After stopping, the state is `Stopped`

### 5.3 Interactive Snippets

Interactive snippets behave differently from normal Runner processes:

- Executed directly via `tea.ExecProcess`
- The child process takes exclusive control of the terminal's stdin/stdout/stderr
- The SnipFly UI temporarily exits the foreground and restores after the child process ends
- Does not go through `Runner`
- Does not support stopping while running
- `r` / `R` does not trigger a restart

### 5.4 Process Exit and Signals

The current implementation handles:

- `SIGINT`
- `SIGTERM`
- `SIGHUP`

Upon receiving a signal, the main program:

1. Calls `Runner.StopAll()`
2. Requests Bubble Tea to quit

After normal program exit, an additional `StopAll()` call is made as a safety net.

---

## 6. Technical Architecture

### 6.1 Dependencies

- `charm.land/bubbletea/v2`: TUI main framework
- `charm.land/bubbles/v2`: Primarily for viewport
- `charm.land/lipgloss/v2`: Styling and layout
- `github.com/creack/pty`: PTY runtime support
- `github.com/spf13/pflag`: Command-line argument parsing

### 6.2 Module Responsibilities

#### `main.go`

Responsible for:

- Parsing CLI arguments
- Computing the actual scan directory
- Scanning snippets
- Creating the `Runner`
- Creating the Bubble Tea program
- Connecting the Runner's output and exit callbacks to `Program.Send(...)`
- Handling exit signals

Important note:

- In the current implementation, `Runner` does not hold a `*tea.Program`
- The `Runner -> TUI` connection works by: `main.go` injecting two closures via `SetCallbacks(...)`, which internally call `p.Send(...)`

#### `internal/snippet`

Responsible for:

- Directory scanning
- Annotation parsing
- Interpreter resolution
- Snippet data model and state definitions

#### `internal/runner`

Responsible for:

- Starting, stopping, and waiting for individual processes
- Multi-process lifecycle management
- Output buffering
- Callback notifications

Core objects:

- `Process`: Wraps a single `exec.Cmd`
- `Runner`: Manages multiple `Process` instances by `FilePath`
- `RingBuffer`: Stores the most recent 1000 lines of output

#### `internal/tui`

Responsible for:

- Root model and message routing
- Left-side list rendering
- Right-side output viewport
- Metadata area rendering
- Bottom status bar
- Exit confirmation dialog
- Output throttling messages

### 6.3 Output Refresh Mechanism

Current output pipeline:

1. Child process stdout/stderr is read line by line by a background goroutine
2. Each line is written to the `RingBuffer`
3. An `OutputMsg` is sent via the `onOutput` callback
4. Upon receiving the first `OutputMsg`, the TUI starts a 50ms throttle tick
5. When the tick expires, the output area is refreshed in bulk

This means:

- Output refresh is not an immediate per-line redraw
- The theoretical maximum refresh rate is approximately 20fps
- High-frequency output reduces TUI redraw pressure

### 6.4 Process Management Strategy

Non-PTY mode:

- Uses `exec.Command`
- Sets `SysProcAttr{Setpgid: true}`
- Reads stdout and stderr separately

PTY mode:

- Uses `pty.Start(...)`
- stdout/stderr are merged into a single PTY fd
- During reading, allows the PTY to return `EIO` after child process exit, treating it as normal

Stop strategy:

- First sends `SIGTERM` to the entire process group
- Waits up to 5 seconds
- Sends `SIGKILL` after timeout
- Falls back to directly terminating the main process if the process group cannot be obtained

Unlike earlier documentation, the current `Stop()` does not proactively close the PTY master before sending the signal; the PTY master is closed after `cmd.Wait()` returns, to unblock the reader.

### 6.5 Output Buffering Strategy

The current buffer implementation is a fixed-size ring buffer:

- Capacity: 1000 lines
- Overwrites the oldest content when full
- `Lines()` returns a chronologically ordered snapshot copy
- `Reset()` clears the output history each time a process is restarted

---

## 7. Release and Version Management

### 7.1 Version Information Injection

`main.go` defines:

- `version = "dev"`
- `commit = "none"`
- `date = "unknown"`

During release builds, real values are injected via `ldflags`, for example:

```bash
go build -ldflags "-X main.version=0.1.0 -X main.commit=abc1234 -X main.date=2025-01-01"
```

The command `snipfly -v` currently outputs:

```text
snipfly version 0.1.0 (commit: abc1234, built: 2025-01-01)
```

### 7.2 GoReleaser

The current `.goreleaser.yaml` configures:

- Multi-platform builds: `darwin`, `linux`, `windows`
- Multi-architecture: `amd64`, `arm64`
- Windows uses `zip`; other platforms use `tar.gz`
- Publishes Homebrew formula to `txperl/homebrew-tap` via `TAP_REPO_TOKEN`

### 7.3 GitHub Actions

The current release workflow triggers on:

- Tag push, pattern `v*`

Workflow steps:

1. `actions/checkout`
2. `actions/setup-go`
3. `goreleaser/goreleaser-action`

### 7.4 Installation Methods

Currently available installation methods for end users:

```bash
brew install txperl/tap/snipfly
```

Or:

```bash
go install github.com/txperl/snipfly@latest
```
