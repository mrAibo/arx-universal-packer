# ARX - Universal Archive Manager

![Version](https://img.shields.io/badge/version-2.0.0-blue.svg)
![License](https://img.shields.io/badge/license-MIT-green.svg)
![Bash](https://img.shields.io/badge/language-bash-orange.svg)
![Platform](https://img.shields.io/badge/platform-linux-lightgrey.svg)

**ARX** (Archive eXtractor) is a modern, powerful, and user-friendly archive manager for Linux. It unifies `tar`, `zip`, `7z`, `zstd`, and `xz` under a single, intuitive interface. No more memorizing complex flags like `tar -czvf` or `tar -xzvf`—ARX handles it all with smart defaults and beautiful output.

![ARX Demo](svgs/ep01_intro.svg)

---

## ✨ Features

### 🚀 Core Capabilities
- **Unified Interface**: One command for all formats (`tar`, `gz`, `bz2`, `xz`, `zst`, `zip`, `7z`).
- **Smart Extraction**: Auto-detects formats and handles "tarbombs" (archives without a root folder).
- **Format Conversion**: Easily convert archives (e.g., `zip` → `tar.zst`) with `arx convert`.
- **Incremental Backups**: Create snapshot-based differential backups to save space.
- **Archive Splitting**: Split large archives into chunks (e.g., for email or FAT32).
- **High Performance**: Multi-threaded compression (auto-detects CPU cores) and **Zstandard** support.

### 🎨 User Experience
- **Beautiful Output**: Color-coded messages, emojis, and real-time progress bars (`pv` integration).
- **Interactive Mode**: A TUI (Text User Interface) wizard for guided operations.
- **GUI Integration**: Context menus for Nautilus, Dolphin, and Thunar ("Compress with ARX").
- **Bash Completion**: Intelligent tab completion for files and options.
- **Safety First**: Dry-run mode, verification (`--verify`), and overwrite protection.

---

## 📦 Installation

### Quick Install (Recommended)

```bash
# Clone repository
git clone https://github.com/yourusername/arx.git
cd arx

# Install system-wide
sudo cp bin/arx /usr/local/bin/
sudo chmod +x /usr/local/bin/arx

# Install man page
sudo cp man/arx.1 /usr/local/share/man/man1/
sudo mandb
```

### Dependencies
ARX works best with these tools:
- **Required**: `bash` (4.4+), `tar`, `gzip`
- **Recommended**: `zstd` (fast compression), `pv` (progress bars), `pigz` (parallel gzip)
- **Optional**: `xz`, `bzip2`, `7z`, `zip`, `dialog` (for interactive mode)

---

## 📖 Usage & Tutorials

### 1. Basic Compression & Extraction
Compress files easily with smart defaults. ARX automatically chooses the best settings.

![Basic Usage](svgs/ep02_basic.svg)

```bash
# Compress a directory (default: tar.gz)
arx -c tar.gz -n backup documents/

# Extract an archive (auto-detects format)
arx backup.tar.gz
```

### 2. Format Conversion
Convert archives from one format to another without manual extraction.

![Conversion](svgs/ep06_convert.svg)

```bash
# Convert zip to tar.zst (Zstandard)
arx convert input.zip to output.tar.zst
```

### 3. Advanced Options
Use exclusions, password protection, and parallel processing.

![Advanced](svgs/ep03_advanced.svg)

```bash
# Exclude files and use max compression
arx -c tar.xz -L 9 -e "*.log" -e "node_modules/" project/

# Password protection (zip/7z)
arx -c zip -p -n secret sensitive_data/
```

### 4. Incremental Backups
Save space by backing up only changed files.

![Incremental](svgs/ep04_incremental.svg)

```bash
# Level 0 (Full Backup)
arx -c tar.gz --incremental backup.snar -n full_backup /data

# Level 1 (Changes Only)
arx -c tar.gz --incremental backup.snar -n inc_backup /data
```

---

## 🎨 Supported Formats

| Format      | Extension(s)        | Compression | Speed      | Ratio           | Notes                |
|-------------|---------------------|-------------|------------|-----------------|----------------------|
| **tar**     | .tar                | None        | ⚡⚡⚡⚡⚡ | -               | Container only       |
| **tar.gz**  | .tar.gz, .tgz       | gzip        | ⚡⚡⚡⚡   | 📦📦📦         | Good balance         |
| **tar.bz2** | .tar.bz2, .tbz2     | bzip2       | ⚡⚡⚡     | 📦📦📦📦       | Better compression   |
| **tar.xz**  | .tar.xz, .txz       | xz/LZMA     | ⚡⚡       | 📦📦📦📦📦     | Best compression     |
| **tar.zst** | .tar.zst            | zstd        | ⚡⚡⚡⚡⚡ | 📦📦📦📦       | Best speed/ratio     |
| **zip**     | .zip                | deflate     | ⚡⚡⚡     | 📦📦📦         | Cross-platform       |
| **7z**      | .7z                 | LZMA2       | ⚡⚡       | 📦📦📦📦📦     | Maximum compression  | 

**Legend:** ⚡ = Speed, 📦 = Compression ratio

---

## 🎓 Advanced Usage

### Pattern-Based Filtering

ARX provides powerful filtering options to include or exclude specific files.

#### Direct Patterns
You can use glob patterns directly in the command line:

```bash
# Exclude temporary files and logs
arx -c tar.gz -e "*.tmp" -e "*.log" -e "temp/" src/

# Include only documentation
arx -c zip -i "*.md" -i "*.txt" -i "*.pdf" docs/

# Complex exclusions
arx -c tar.gz \
  -e "node_modules/" \
  -e ".git/" \
  -e "*.lock" \
  -e "build/" \
  -e "dist/" \
  project/
```

#### Pattern Files
For complex projects, you can list patterns in a file:

```bash
# Create a patterns file
echo "-node_modules/" > .arxignore
echo "-*.log" >> .arxignore
echo "+src/" >> .arxignore

# Use it with -f
arx -c tar.gz -f .arxignore -n project-backup .
```

---

## ⚙️ Configuration

Create `~/.config/arx/config` to set your persistent preferences.

### Configuration Parameters

| Parameter | Description | Default | Example |
|-----------|-------------|---------|---------|
| `default_format` | Default archive format | `tar.gz` | `tar.zst` |
| `default_level` | Compression level (0-9) | `3` | `9` |
| `default_exclude` | Global exclude patterns | (empty) | `*.tmp *.log .git/` |
| `default_jobs` | Number of threads (0=auto) | `0` | `4` |
| `use_spinner` | Use spinner if pv missing | `true` | `false` |

### Example Config File
```ini
# ~/.config/arx/config

# Use Zstandard by default for speed
default_format = tar.zst

# Exclude git and temp files globally
default_exclude = .git/ *.tmp *.swp __pycache__/

# Use 4 threads for compression
default_jobs = 4
```

---

## 🖥️ Interactive Mode & GUI

### Interactive TUI
Simply run `arx` without arguments to start the interactive wizard.
It uses `dialog` (if installed) or a text-based menu to guide you through:
1.  **Mode Selection** (Compress, Extract, List, Convert)
2.  **Format Selection** (tar.gz, zip, etc.)
3.  **File Selection** (with path completion)
4.  **Options** (Password, Split, etc.)

![Interactive Demo](svgs/ep01_intro.svg)

### GUI Integration
ARX integrates directly into your file manager's context menu.
- **Right-click** > **Compress with ARX**
- **Right-click** > **Extract with ARX**

Supported File Managers:
- **Nautilus** (GNOME)
- **Dolphin** (KDE)
- **Thunar** (XFCE)

---

## 🚀 Progress Bar & Performance

ARX automatically detects if `pv` (Pipe Viewer) is installed to show beautiful progress bars.

```
ℹ Creating backup.tar.gz (2.4 GB)
2.40GB 0:01:15 [32.0MB/s] [==================>] 100%
```

If `pv` is missing, it falls back to a sleek spinner animation so you still know it's working.

**Performance Tips:**
- **Multithreading**: ARX automatically uses all available CPU cores for `xz`, `zstd`, and `pigz`.
- **Fastest Backup**: Use `tar.zst` (Zstandard) for the best speed-to-ratio balance.

---

## 🐛 Troubleshooting

### Common Issues

#### "Command not found"
**Solution**: Install missing dependencies.
```bash
# Ubuntu/Debian
sudo apt install tar gzip zstd pv

# Fedora/RHEL
sudo dnf install tar gzip zstd pv
```

#### "Permission denied"
**Solution**: Check file permissions or use `sudo` if necessary. ARX preserves permissions by default.

#### "Archive corrupted"
**Solution**:
1. Check disk space: `df -h`
2. Verify archive: `arx --verify archive.tar.gz`

#### Autocompletion not working
**Solution**: Ensure you sourced the script or installed the completion file.
```bash
source /path/to/arx
# or
source /etc/bash_completion.d/arx
```

---

## 🤝 Contributing

Contributions are welcome! See [CONTRIBUTING.md](CONTRIBUTING.md) for details.

1. Fork the repo
2. Create a feature branch
3. Submit a Pull Request

## 📝 License

Distributed under the MIT License. See [LICENSE](LICENSE).

---

**Made with ❤️ for the Linux community.**
