# PAC.SH - Quick Reference Card

## 📋 Most Common Commands

```bash
# Compress files
pac -c tar.gz -n backup files/

# Extract archive  
pac archive.tar.gz

# List contents
pac -l archive.zip

# Compress with verification
pac -c tar.xz -n data --verify folder/
```

## 🎯 Cheat Sheet

### Compression
| Command | Effect |
|---------|--------|
| `pac -c tar.gz -n NAME FILES` | Create gzip archive |
| `pac -c tar.xz -n NAME FILES` | Create xz archive (best ratio) |
| `pac -c zip -n NAME FILES` | Create zip archive |
| `pac -c tar.gz -L 9 -n NAME FILES` | Maximum compression |
| `pac -c tar.xz -j 8 -n NAME FILES` | Use 8 CPU cores |

### Extraction
| Command | Effect |
|---------|--------|
| `pac ARCHIVE` | Extract to current dir |
| `pac -t /path ARCHIVE` | Extract to specific dir |
| `pac -v ARCHIVE` | Extract with details |

### Filtering
| Command | Effect |
|---------|--------|
| `pac -c tar.gz -n NAME -e "*.tmp" FILES` | Exclude .tmp files |
| `pac -c tar.gz -n NAME -i "*.txt" FILES` | Include only .txt |
| `pac -c tar.gz -e "*.log" -e "cache" FILES` | Multiple excludes |

### Useful Flags
| Flag | Effect |
|------|--------|
| `--verify` | Verify after compression |
| `--dry-run` | Preview without executing |
| `-q` | Quiet (no output) |
| `-v` | Verbose (detailed) |
| `-d` | Delete originals after |
| `--no-confirm` | Skip prompts |

## 🚀 Quick Start

### Installation
```bash
# Make executable
chmod +x pac.sh

# Copy to PATH
sudo cp pac.sh /usr/local/bin/pac

# Enable bash completion (add to ~/.bashrc)
source /path/to/pac.sh
```

### First Use
```bash
# Show help
pac -h

# Show version
pac --version

# Test it
echo "test" > file.txt
pac -c tar.gz -n test file.txt
pac -l test.tar.gz
```

## 💡 Pro Tips

### Fastest Compression
```bash
pac -c tar -n fast big_files/
# OR
pac -c tar.zst -L 1 -n fast big_files/
```

### Best Compression
```bash
pac -c tar.xz -L 9 -n small big_files/
```

### Balanced (Recommended)
```bash
pac -c tar.xz -n backup files/
```

### Parallel Compression
```bash
pac -c tar.xz -j $(nproc) -n fast big_dir/
```

### Safe Backup
```bash
pac -c tar.xz -n backup --verify important_data/
```

## 🎨 Format Comparison

| Format | Speed | Ratio | Use Case |
|--------|-------|-------|----------|
| tar | ⭐⭐⭐⭐⭐ | - | No compression needed |
| tar.gz | ⭐⭐⭐⭐ | ⭐⭐⭐ | Default, good compatibility |
| tar.bz2 | ⭐⭐⭐ | ⭐⭐⭐⭐ | Better than gzip |
| tar.xz | ⭐⭐ | ⭐⭐⭐⭐⭐ | **Best ratio (recommended)** |
| tar.zst | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐ | **Fast + good ratio** |
| zip | ⭐⭐⭐⭐ | ⭐⭐⭐ | Universal compatibility |
| 7z | ⭐⭐ | ⭐⭐⭐⭐⭐ | Excellent ratio |

## 🔧 Troubleshooting

### "Fehlende Abhängigkeiten"
```bash
# Fedora
sudo dnf install tar gzip xz zip p7zip

# Ubuntu/Debian
sudo apt install tar gzip xz-utils zip p7zip-full
```

### "pv nicht installiert"
```bash
# Optional - for progress bars
sudo dnf install pv    # Fedora
sudo apt install pv    # Ubuntu
```

### Verification Failed
- Check disk space
- Try different format
- Check source files

## 📝 Examples by Use Case

### Daily Backup
```bash
pac -c tar.xz -n "backup-$(date +%Y%m%d)" \
    --verify ~/documents ~/projects
```

### Compress Logs
```bash
pac -c tar.gz -n logs -i "*.log" /var/log/
```

### Source Code Archive
```bash
pac -c tar.gz -n myapp-v1.0 \
    -e ".git" -e "*.pyc" -e "__pycache__" \
    myapp/
```

### Quick Test Archive
```bash
pac -c zip -n test --dry-run files/  # Preview
pac -c zip -n test files/             # Execute
```

---

**Need more help?** 
- Full docs: `cat README.md`
- Man page: `man ./pac.1`  
- Help: `pac -h`
- Examples: Error messages include hints!
