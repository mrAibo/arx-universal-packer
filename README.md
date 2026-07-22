# ARX — Universal Archive Commander

![License](https://img.shields.io/badge/license-MIT-green.svg)
![Language](https://img.shields.io/badge/language-Go-00ADD8.svg)
![Platform](https://img.shields.io/badge/platform-Linux-lightgrey.svg)

**ARX** is a keyboard-driven, dual-pane file and archive manager for Linux. Its interface follows the familiar Midnight Commander workflow while adding archive creation, extraction, testing, conversion, safe file operations, progress reporting, cancellation, Trash integration, and undo.

The repository also contains the original Bash-based archive utility. The Go commander in `arx-go/` is the primary interactive application.

---

## Highlights

- Dual-pane terminal file manager
- Keyboard and mouse navigation
- Archive creation, extraction, viewing, testing, editing, and conversion
- Support for `tar`, `tar.gz`, `tar.bz2`, `tar.xz`, `tar.zst`, `zip`, and `7z`
- Midnight Commander-style copy and move conflict dialogs
- Safe copy with temporary targets, atomic completion, and rollback
- Safe move with cross-filesystem (`EXDEV`) fallback
- Desktop Trash integration instead of irreversible deletion
- `Ctrl+Z` restore for the latest Trash batch in the current session
- Operation progress with cancellation
- Protection against unintended overwrites
- Tests for normal paths, conflicts, rollback, cross-filesystem behavior, and restore safety

---

## Interface

ARX opens with two filesystem panels. The active panel is the source for most operations, while the opposite panel is used as the default destination for copy, move, extraction, and related commands.

### Keyboard shortcuts

| Key | Action |
|---|---|
| `Tab` | Switch active panel |
| Arrow keys | Navigate files and directories |
| `Enter` | Open a directory or archive |
| `Backspace` | Go to the parent directory |
| `F1` | Help |
| `F2` | Create archive |
| `F3` | View file or archive member |
| `F4` | Test archive |
| `F5` | Copy selected item(s) |
| `F6` | Move or rename selected item(s) |
| `Alt+F6` | Convert archive |
| `F7` | Create directory |
| `F8` | Move selected item(s) to Trash |
| `Ctrl+Z` | Restore the latest Trash batch |
| `F9` | Menu |
| `F10` | Quit |
| `Esc` | Close a dialog or cancel a running operation |

The exact dialog options depend on the current selection. For example, archive actions appear when an archive is selected, while normal files and directories expose filesystem operations.

---

## Safe filesystem operations

ARX treats file operations as recoverable transactions wherever practical.

### Copy

Copy operations write through temporary targets and complete only after the destination is ready. If an operation fails or is cancelled, ARX removes incomplete output and preserves the original source.

When a destination already exists, ARX presents an MC-style conflict dialog instead of silently overwriting it.

### Move

Moves use the native filesystem rename operation when possible. When source and destination are on different filesystems, ARX automatically falls back to a guarded copy-and-remove flow.

The source is removed only after the destination has been completed successfully. Failure and cancellation paths retain the source and clean up incomplete output.

### Trash and undo

`F8` sends files and directories to the freedesktop-compatible Trash location instead of permanently deleting them.

After a successful Trash operation, `Ctrl+Z` can restore the latest batch from the current ARX session. Restore never overwrites a path that has since been recreated. Conflicting entries remain safely in Trash and can be retried later.

The undo history is intentionally limited to the latest filesystem Trash operation in the running ARX process. It does not restore archive-member deletions or arbitrary items placed in Trash by other applications.

---

## Archive support

| Format | Common extensions | External tool |
|---|---|---|
| TAR | `.tar` | `tar` |
| Gzip TAR | `.tar.gz`, `.tgz` | `tar`, `gzip` |
| Bzip2 TAR | `.tar.bz2`, `.tbz2` | `tar`, `bzip2` |
| XZ TAR | `.tar.xz`, `.txz` | `tar`, `xz` |
| Zstandard TAR | `.tar.zst`, `.tzst` | `tar`, `zstd` |
| ZIP | `.zip` | `zip`, `unzip` |
| 7-Zip | `.7z` | `7z` or `7zz` |

Only the tools required for the formats you use need to be installed. ARX reports a clear error when a required helper program is unavailable.

---

## Installation

### Use the included Linux binary

The repository contains a prebuilt binary at:

```text
dist/arx-go
```

After cloning the repository:

```bash
git clone https://github.com/mrAibo/arx-universal-packer.git
cd arx-universal-packer
chmod +x dist/arx-go
./dist/arx-go
```

To install it system-wide:

```bash
sudo install -Dm755 dist/arx-go /usr/local/bin/arx
arx
```

### Build from source

The Go module is located in `arx-go/`.

Requirements:

- Linux
- Go version declared in `arx-go/go.mod`

Build:

```bash
git clone https://github.com/mrAibo/arx-universal-packer.git
cd arx-universal-packer/arx-go
go build -o ../dist/arx-go .
../dist/arx-go
```

---

## Install archive helpers

### Debian / Ubuntu

```bash
sudo apt install tar gzip bzip2 xz-utils zstd zip unzip p7zip-full
```

### Fedora / RHEL

```bash
sudo dnf install tar gzip bzip2 xz zstd zip unzip p7zip
```

### Arch Linux

```bash
sudo pacman -S tar gzip bzip2 xz zstd zip unzip 7zip
```

Package names can differ between distributions. Install only the helpers needed for your preferred archive formats.

---

## Development

Run the Go tests from the module directory:

```bash
cd arx-go
go test ./...
```

Run them with the race detector:

```bash
go test -race ./...
```

Build the distribution binary:

```bash
go build -o ../dist/arx-go .
```

The project includes focused coverage for copy and move conflicts, cancellation, rollback, cross-filesystem moves, Trash metadata, restore round trips, overwrite protection, and partial restore behavior.

---

## Project layout

```text
arx-universal-packer/
├── arx-go/        Go-based dual-pane commander
├── bin/           Original Bash command-line utility
├── dist/          Built distribution artifacts
├── man/           Manual pages for the legacy CLI
├── svgs/          Demo assets
└── tests/         Repository-level tests
```

---

## Legacy Bash CLI

The original Bash archive utility remains available in `bin/arx`. It provides command-line archive creation, extraction, conversion, filtering, incremental backups, splitting, and desktop integration.

Use the Go application for the dual-pane commander experience. Use the Bash utility when scripting or when its specialized command-line options are required.

---

## Current scope

ARX currently targets Linux terminals and local filesystems. Remote filesystems mounted by the operating system can be used like normal directories, but ARX does not yet provide a native SFTP, FTP, or cloud-storage client.

Potential future work includes background jobs, pause and resume, Linux packages, broader terminal interaction tests, localization, and native remote filesystem connections.

---

## Troubleshooting

### An archive action reports a missing command

Install the helper listed in the error message. For example, `.7z` support requires `7z` or `7zz`, while `.tar.zst` requires `tar` and `zstd`.

### A destination already exists

Use the conflict dialog to choose the appropriate action. ARX does not silently replace existing paths.

### A restore does not overwrite an existing path

This is intentional. ARX leaves the affected item in Trash when its original path has been recreated. Rename or remove the conflicting path, then retry while the undo batch remains available.

### Permission denied

Verify permissions for the source, destination, and Trash location. Running a file manager with `sudo` is generally discouraged because it can create root-owned files in the user's home directory.

---

## Contributing

Contributions and focused bug reports are welcome.

1. Fork the repository.
2. Create a small, clearly scoped branch.
3. Add or update tests for behavioral changes.
4. Run `go test -race ./...` in `arx-go/`.
5. Open a pull request describing the behavior and safety implications.

Please prefer small changes, existing project patterns, Go's standard library, and fixes that address the underlying cause.

---

## License

Distributed under the MIT License. See [LICENSE](LICENSE).
