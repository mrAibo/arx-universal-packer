# ARX - Universeller Archiv-Manager

![Version](https://img.shields.io/badge/version-2.0.0-blue.svg)
![Lizenz](https://img.shields.io/badge/license-MIT-green.svg)
![Bash](https://img.shields.io/badge/language-bash-orange.svg)
![Plattform](https://img.shields.io/badge/platform-linux-lightgrey.svg)

**ARX** (Archive eXtractor) ist ein moderner, leistungsstarker und benutzerfreundlicher Archiv-Manager für Linux. Er vereint `tar`, `zip`, `7z`, `zstd` und `xz` unter einer einzigen, intuitiven Oberfläche. Schluss mit dem Auswendiglernen komplexer Befehle wie `tar -czvf` oder `tar -xzvf` – ARX erledigt alles mit intelligenten Standards und übersichtlicher Ausgabe.

![ARX Demo](tutorials/svgs/ep01_intro.svg)

---

## ✨ Funktionen

### 🚀 Kernfunktionen
- **Einheitliche Oberfläche**: Ein Befehl für alle Formate (`tar`, `gz`, `bz2`, `xz`, `zst`, `zip`, `7z`).
- **Intelligentes Entpacken**: Erkennt Formate automatisch und behandelt "Tarbombs" (Archive ohne Wurzelverzeichnis).
- **Format-Konvertierung**: Konvertieren Sie Archive einfach (z.B. `zip` → `tar.zst`) mit `arx convert`.
- **Inkrementelle Backups**: Erstellen Sie platzsparende, Snapshot-basierte Backups.
- **Archiv-Splitting**: Teilen Sie große Archive in Stücke (z.B. für E-Mail oder FAT32).
- **Hohe Leistung**: Multithread-Kompression (automatische Kern-Erkennung) und **Zstandard**-Support.

### 🎨 Benutzererfahrung
- **Schöne Ausgabe**: Farbcodierte Nachrichten, Emojis und Echtzeit-Fortschrittsbalken (`pv`-Integration).
- **Interaktiver Modus**: Ein TUI (Text User Interface) Assistent für geführte Operationen.
- **GUI-Integration**: Kontextmenüs für Nautilus, Dolphin und Thunar ("Mit ARX komprimieren").
- **Bash-Completion**: Intelligente Tab-Vervollständigung für Dateien und Optionen.
- **Sicherheit**: Dry-Run-Modus, Verifizierung (`--verify`) und Überschreibschutz.

---

## 📦 Installation

### Schnellinstallation (Empfohlen)

```bash
# Repository klonen
git clone https://github.com/yourusername/arx.git
cd arx

# Systemweit installieren
sudo cp bin/arx /usr/local/bin/
sudo chmod +x /usr/local/bin/arx

# Man-Page installieren
sudo cp man/arx.1 /usr/local/share/man/man1/
sudo mandb
```

### Abhängigkeiten
ARX funktioniert am besten mit diesen Tools:
- **Erforderlich**: `bash` (4.4+), `tar`, `gzip`
- **Empfohlen**: `zstd` (schnelle Kompression), `pv` (Fortschrittsbalken), `pigz` (paralleles gzip)
- **Optional**: `xz`, `bzip2`, `7z`, `zip`, `dialog` (für interaktiven Modus)

---

## 📖 Verwendung & Tutorials

### 1. Einfaches Komprimieren & Entpacken
Komprimieren Sie Dateien einfach mit intelligenten Standards.

![Basis-Nutzung](tutorials/svgs/ep02_basic.svg)

```bash
# Verzeichnis komprimieren (Standard: tar.gz)
arx -c tar.gz -n backup dokumente/

# Archiv entpacken (Format wird automatisch erkannt)
arx backup.tar.gz
```

### 2. Format-Konvertierung
Konvertieren Sie Archive von einem Format in ein anderes ohne manuelles Entpacken.

![Konvertierung](tutorials/svgs/ep06_convert.svg)

```bash
# Konvertiere zip zu tar.zst (Zstandard)
arx convert eingabe.zip to ausgabe.tar.zst
```

### 3. Erweiterte Optionen
Nutzen Sie Ausschlüsse, Passwortschutz und parallele Verarbeitung.

![Erweitert](tutorials/svgs/ep03_advanced.svg)

```bash
# Dateien ausschließen und maximale Kompression nutzen
arx -c tar.xz -L 9 -e "*.log" -e "node_modules/" projekt/

# Passwortschutz (zip/7z)
arx -c zip -p -n geheim sensible_daten/
```

### 4. Inkrementelle Backups
Sparen Sie Platz, indem Sie nur geänderte Dateien sichern.

![Inkrementell](tutorials/svgs/ep04_incremental.svg)

```bash
# Level 0 (Vollbackup)
arx -c tar.gz --incremental backup.snar -n voll_backup /daten

# Level 1 (Nur Änderungen)
arx -c tar.gz --incremental backup.snar -n inc_backup /daten
```

---

## 🎨 Unterstützte Formate

| Format      | Endung(en)          | Kompression | Tempo      | Ratio           | Notizen              |
|-------------|---------------------|-------------|------------|-----------------|----------------------|
| **tar**     | .tar                | Keine       | ⚡⚡⚡⚡⚡ | -               | Nur Container        |
| **tar.gz**  | .tar.gz, .tgz       | gzip        | ⚡⚡⚡⚡   | 📦📦📦         | Gute Balance         |
| **tar.bz2** | .tar.bz2, .tbz2     | bzip2       | ⚡⚡⚡     | 📦📦📦📦       | Bessere Kompression  |
| **tar.xz**  | .tar.xz, .txz       | xz/LZMA     | ⚡⚡       | 📦📦📦📦📦     | Beste Kompression    |
| **tar.zst** | .tar.zst            | zstd        | ⚡⚡⚡⚡⚡ | 📦📦📦📦       | Bestes Tempo/Ratio   |
| **zip**     | .zip                | deflate     | ⚡⚡⚡     | 📦📦📦         | Cross-Platform       |
| **7z**      | .7z                 | LZMA2       | ⚡⚡       | 📦📦📦📦📦     | Max. Kompression     | 

**Legende:** ⚡ = Geschwindigkeit, 📦 = Kompressionsrate

---

## 🎓 Erweiterte Nutzung

### Muster-basierte Filterung

ARX bietet leistungsstarke Filteroptionen zum Ein- oder Ausschließen bestimmter Dateien.

#### Direkte Muster
Sie können Glob-Muster direkt in der Kommandozeile verwenden:

```bash
# Temporäre Dateien und Logs ausschließen
arx -c tar.gz -e "*.tmp" -e "*.log" -e "temp/" src/

# Nur Dokumentation einschließen
arx -c zip -i "*.md" -i "*.txt" -i "*.pdf" docs/

# Komplexe Ausschlüsse
arx -c tar.gz \
  -e "node_modules/" \
  -e ".git/" \
  -e "*.lock" \
  -e "build/" \
  -e "dist/" \
  project/
```

#### Muster-Dateien
Für komplexe Projekte können Sie Muster in einer Datei auflisten:

```bash
# Musterdatei erstellen
echo "-node_modules/" > .arxignore
echo "-*.log" >> .arxignore
echo "+src/" >> .arxignore

# Mit -f verwenden
arx -c tar.gz -f .arxignore -n projekt-backup .
```

---

## ⚙️ Konfiguration

Erstellen Sie `~/.config/arx/config`, um Ihre Einstellungen dauerhaft zu speichern.

### Konfigurationsparameter

| Parameter | Beschreibung | Standard | Beispiel |
|-----------|--------------|----------|----------|
| `default_format` | Standard-Archivformat | `tar.gz` | `tar.zst` |
| `default_level` | Kompressionslevel (0-9) | `3` | `9` |
| `default_exclude` | Globale Ausschlussmuster | (leer) | `*.tmp *.log .git/` |
| `default_jobs` | Anzahl Threads (0=auto) | `0` | `4` |
| `use_spinner` | Spinner nutzen wenn pv fehlt | `true` | `false` |

### Beispiel-Konfigurationsdatei
```ini
# ~/.config/arx/config

# Zstandard als Standard für Geschwindigkeit
default_format = tar.zst

# Git und temporäre Dateien global ausschließen
default_exclude = .git/ *.tmp *.swp __pycache__/

# 4 Threads für Kompression nutzen
default_jobs = 4
```

---

## 🖥️ Interaktiver Modus & GUI

### Interaktives TUI
Führen Sie einfach `arx` ohne Argumente aus, um den interaktiven Assistenten zu starten.
Er verwendet `dialog` (falls installiert) oder ein textbasiertes Menü:
1.  **Modus-Auswahl** (Komprimieren, Entpacken, Auflisten, Konvertieren)
2.  **Format-Auswahl** (tar.gz, zip, etc.)
3.  **Datei-Auswahl** (mit Pfadvervollständigung)
4.  **Optionen** (Passwort, Splitting, etc.)

![Interaktive Demo](tutorials/svgs/ep01_intro.svg)

### GUI-Integration
ARX integriert sich direkt in das Kontextmenü Ihres Dateimanagers.
- **Rechtsklick** > **Mit ARX komprimieren**
- **Rechtsklick** > **Mit ARX entpacken**

Unterstützte Dateimanager:
- **Nautilus** (GNOME)
- **Dolphin** (KDE)
- **Thunar** (XFCE)

---

## 🚀 Fortschrittsbalken & Leistung

ARX erkennt automatisch, ob `pv` (Pipe Viewer) installiert ist, um schöne Fortschrittsbalken anzuzeigen.

```
ℹ Creating backup.tar.gz (2.4 GB)
2.40GB 0:01:15 [32.0MB/s] [==================>] 100%
```

Wenn `pv` fehlt, wird eine elegante Spinner-Animation verwendet.

**Leistungstipps:**
- **Multithreading**: ARX nutzt automatisch alle verfügbaren CPU-Kerne für `xz`, `zstd` und `pigz`.
- **Schnellstes Backup**: Verwenden Sie `tar.zst` (Zstandard) für das beste Verhältnis von Geschwindigkeit und Größe.

---

## 🐛 Fehlerbehebung

### Häufige Probleme

#### "Command not found"
**Lösung**: Installieren Sie fehlende Abhängigkeiten.
```bash
# Ubuntu/Debian
sudo apt install tar gzip zstd pv

# Fedora/RHEL
sudo dnf install tar gzip zstd pv
```

#### "Permission denied"
**Lösung**: Überprüfen Sie Dateiberechtigungen oder nutzen Sie `sudo`. ARX behält Berechtigungen standardmäßig bei.

#### "Archive corrupted"
**Lösung**:
1. Speicherplatz prüfen: `df -h`
2. Archiv verifizieren: `arx --verify archiv.tar.gz`

#### Autovervollständigung funktioniert nicht
**Lösung**: Stellen Sie sicher, dass das Skript gesourct oder die Completion-Datei installiert ist.
```bash
source /pfad/zu/arx
# oder
source /etc/bash_completion.d/arx
```

---

## 🤝 Mitwirken

Beiträge sind willkommen! Siehe [CONTRIBUTING.md](CONTRIBUTING.md) für Details.

1. Forken Sie das Repo
2. Erstellen Sie einen Feature-Branch
3. Senden Sie einen Pull Request

## 📝 Lizenz

Veröffentlicht unter der MIT-Lizenz. Siehe [LICENSE](LICENSE).

---

**Gemacht mit ❤️ für die Linux-Community.**
