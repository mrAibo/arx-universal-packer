package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type paneMode int

const (
	paneFilesystem paneMode = iota
	paneArchive
)

type fileEntry struct {
	Name      string
	Path      string
	IsDir     bool
	IsArchive bool
	Size      int64
	ModTime   time.Time
}

type pane struct {
	mode          paneMode
	path          string
	archivePath   string
	archivePrefix string
	archiveItems  []string
	entries       []fileEntry
	cursor        int
	offset        int
	showHidden    bool
	err           string
}

func newPane(path string) pane {
	absolute, err := filepath.Abs(path)
	if err != nil {
		absolute = path
	}
	p := pane{mode: paneFilesystem, path: absolute}
	if err := p.reload(); err != nil {
		p.err = err.Error()
	}
	return p
}

func (p *pane) reload() error {
	selectedName := ""
	if selected, ok := p.selected(); ok {
		selectedName = selected.Name
	}

	var err error
	if p.mode == paneArchive {
		err = p.loadArchiveView()
	} else {
		err = p.loadDirectory()
	}
	if err != nil {
		p.err = err.Error()
		return err
	}
	p.err = ""
	p.selectName(selectedName)
	return nil
}

func (p *pane) loadDirectory() error {
	items, err := os.ReadDir(p.path)
	if err != nil {
		return fmt.Errorf("read %s: %w", p.path, err)
	}

	entries := make([]fileEntry, 0, len(items)+1)
	if !isFilesystemRoot(p.path) {
		entries = append(entries, fileEntry{Name: "..", Path: filepath.Dir(p.path), IsDir: true, Size: -1})
	}
	for _, item := range items {
		if !p.showHidden && strings.HasPrefix(item.Name(), ".") {
			continue
		}
		info, infoErr := item.Info()
		entry := fileEntry{
			Name:      item.Name(),
			Path:      filepath.Join(p.path, item.Name()),
			IsDir:     item.IsDir(),
			IsArchive: !item.IsDir() && DetectFormat(strings.ToLower(item.Name())) != "unknown",
			Size:      -1,
		}
		if infoErr == nil {
			entry.Size = info.Size()
			entry.ModTime = info.ModTime()
		}
		entries = append(entries, entry)
	}

	sortEntries(entries)
	p.entries = entries
	p.clampCursor()
	return nil
}

func (p *pane) loadArchive(path string) error {
	items, err := readArchiveItems(path)
	if err != nil {
		return err
	}
	p.mode = paneArchive
	p.archivePath = path
	p.archivePrefix = ""
	p.archiveItems = items
	p.cursor = 0
	p.offset = 0
	if err := p.loadArchiveView(); err != nil {
		p.mode = paneFilesystem
		p.archivePath = ""
		p.archiveItems = nil
		return err
	}
	return nil
}

func (p *pane) loadArchiveView() error {
	if p.archivePath == "" {
		return fmt.Errorf("archive path is empty")
	}
	entries := buildArchiveEntries(p.archiveItems, p.archivePrefix)
	p.entries = append([]fileEntry{{Name: "..", IsDir: true, Size: -1}}, entries...)
	p.clampCursor()
	return nil
}

func (p *pane) selected() (fileEntry, bool) {
	if p.cursor < 0 || p.cursor >= len(p.entries) {
		return fileEntry{}, false
	}
	return p.entries[p.cursor], true
}

func (p *pane) move(delta, rows int) {
	if len(p.entries) == 0 {
		p.cursor = 0
		p.offset = 0
		return
	}
	p.cursor += delta
	if p.cursor < 0 {
		p.cursor = 0
	}
	if p.cursor >= len(p.entries) {
		p.cursor = len(p.entries) - 1
	}
	p.ensureVisible(rows)
}

func (p *pane) ensureVisible(rows int) {
	if rows < 1 {
		rows = 1
	}
	p.clampCursor()
	if p.cursor < p.offset {
		p.offset = p.cursor
	}
	if p.cursor >= p.offset+rows {
		p.offset = p.cursor - rows + 1
	}
	maxOffset := len(p.entries) - rows
	if maxOffset < 0 {
		maxOffset = 0
	}
	if p.offset > maxOffset {
		p.offset = maxOffset
	}
	if p.offset < 0 {
		p.offset = 0
	}
}

func (p pane) visibleEntries(rows int) []fileEntry {
	if rows <= 0 || p.offset >= len(p.entries) {
		return nil
	}
	end := p.offset + rows
	if end > len(p.entries) {
		end = len(p.entries)
	}
	return p.entries[p.offset:end]
}

func (p *pane) openSelected() error {
	entry, ok := p.selected()
	if !ok {
		return nil
	}
	if entry.Name == ".." {
		return p.goUp()
	}

	if p.mode == paneArchive {
		if !entry.IsDir {
			return nil
		}
		p.archivePrefix = joinArchivePrefix(p.archivePrefix, entry.Name)
		p.cursor = 0
		p.offset = 0
		return p.loadArchiveView()
	}

	if entry.IsDir {
		p.path = entry.Path
		p.cursor = 0
		p.offset = 0
		return p.loadDirectory()
	}
	if entry.IsArchive {
		return p.loadArchive(entry.Path)
	}
	return nil
}

func (p *pane) goUp() error {
	if p.mode == paneArchive {
		if p.archivePrefix != "" {
			old := archiveBase(p.archivePrefix)
			p.archivePrefix = archiveParent(p.archivePrefix)
			if err := p.loadArchiveView(); err != nil {
				return err
			}
			p.selectName(old)
			return nil
		}
		archiveName := filepath.Base(p.archivePath)
		p.mode = paneFilesystem
		p.archivePath = ""
		p.archivePrefix = ""
		p.archiveItems = nil
		if err := p.loadDirectory(); err != nil {
			return err
		}
		p.selectName(archiveName)
		return nil
	}

	if isFilesystemRoot(p.path) {
		return nil
	}
	oldName := filepath.Base(p.path)
	p.path = filepath.Dir(p.path)
	p.cursor = 0
	p.offset = 0
	if err := p.loadDirectory(); err != nil {
		return err
	}
	p.selectName(oldName)
	return nil
}

func (p *pane) createDirectory() error {
	if p.mode != paneFilesystem {
		return fmt.Errorf("cannot create a directory inside an archive yet")
	}
	name := "new-folder"
	for i := 1; ; i++ {
		candidate := filepath.Join(p.path, name)
		if _, err := os.Stat(candidate); os.IsNotExist(err) {
			if err := os.Mkdir(candidate, 0o755); err != nil {
				return err
			}
			if err := p.loadDirectory(); err != nil {
				return err
			}
			p.selectName(name)
			return nil
		}
		name = fmt.Sprintf("new-folder-%d", i)
	}
}

func (p pane) location() string {
	if p.mode == paneArchive {
		location := filepath.Base(p.archivePath) + ":/"
		if p.archivePrefix != "" {
			location += p.archivePrefix
		}
		return location
	}
	return p.path
}

func (p pane) selectedDescription() string {
	entry, ok := p.selected()
	if !ok {
		return p.location()
	}
	if entry.Name == ".." {
		return "Parent directory"
	}
	kind := "file"
	if entry.IsDir {
		kind = "directory"
	} else if entry.IsArchive {
		kind = "archive"
	}
	if entry.Size >= 0 && !entry.IsDir {
		return fmt.Sprintf("%s · %s · %s", entry.Name, kind, formatSize(entry.Size))
	}
	return fmt.Sprintf("%s · %s", entry.Name, kind)
}

func (p *pane) selectName(name string) {
	if name == "" {
		p.clampCursor()
		return
	}
	for i, entry := range p.entries {
		if entry.Name == name {
			p.cursor = i
			p.offset = 0
			return
		}
	}
	p.clampCursor()
}

func (p *pane) clampCursor() {
	if len(p.entries) == 0 {
		p.cursor = 0
		p.offset = 0
		return
	}
	if p.cursor < 0 {
		p.cursor = 0
	}
	if p.cursor >= len(p.entries) {
		p.cursor = len(p.entries) - 1
	}
}

func sortEntries(entries []fileEntry) {
	start := 0
	if len(entries) > 0 && entries[0].Name == ".." {
		start = 1
	}
	sort.SliceStable(entries[start:], func(i, j int) bool {
		left := entries[start+i]
		right := entries[start+j]
		if left.IsDir != right.IsDir {
			return left.IsDir
		}
		return strings.ToLower(left.Name) < strings.ToLower(right.Name)
	})
}

func readArchiveItems(path string) ([]string, error) {
	format := DetectFormat(strings.ToLower(path))
	var output string
	var err error

	switch format {
	case "tar":
		output, err = runCapture("tar", "-tf", path)
	case "tar.gz":
		output, err = runCapture("tar", "-tzf", path)
	case "tar.bz2":
		output, err = runCapture("tar", "-tjf", path)
	case "tar.xz":
		output, err = runCapture("tar", "-tJf", path)
	case "tar.zst":
		output, err = runCapture("tar", "--zstd", "-tf", path)
	case "zip":
		output, err = runCapture("unzip", "-Z1", path)
	case "7z":
		output, err = runCapture("7z", "l", "-ba", "-slt", path)
	default:
		return nil, fmt.Errorf("unsupported archive format: %s", path)
	}
	if err != nil {
		if strings.TrimSpace(output) != "" {
			return nil, fmt.Errorf("cannot list archive: %s", strings.TrimSpace(output))
		}
		return nil, fmt.Errorf("cannot list archive: %w", err)
	}

	if format == "7z" {
		return parse7zPaths(output, path), nil
	}
	return parseArchiveLines(output), nil
}

func parseArchiveLines(output string) []string {
	var paths []string
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		value := normalizeArchivePath(scanner.Text())
		if value != "" {
			paths = append(paths, value)
		}
	}
	return paths
}

func parse7zPaths(output, archivePath string) []string {
	archiveBaseName := filepath.Base(archivePath)
	archiveNormalized := normalizeArchivePath(archivePath)
	var paths []string
	var current string
	currentIsDir := false

	flush := func() {
		value := normalizeArchivePath(current)
		if value == "" || value == archiveNormalized || value == archiveBaseName {
			current = ""
			currentIsDir = false
			return
		}
		if currentIsDir {
			value += "/"
		}
		paths = append(paths, value)
		current = ""
		currentIsDir = false
	}

	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()
		switch {
		case strings.HasPrefix(line, "Path = "):
			flush()
			current = strings.TrimPrefix(line, "Path = ")
		case strings.HasPrefix(line, "Attributes = "):
			attributes := strings.TrimSpace(strings.TrimPrefix(line, "Attributes = "))
			currentIsDir = strings.HasPrefix(attributes, "D")
		case strings.TrimSpace(line) == "":
			flush()
		}
	}
	flush()
	return paths
}

func buildArchiveEntries(paths []string, prefix string) []fileEntry {
	prefix = normalizeArchivePrefix(prefix)
	type aggregate struct {
		name  string
		isDir bool
	}
	items := make(map[string]aggregate)

	for _, original := range paths {
		path := normalizeArchivePath(original)
		if path == "" {
			continue
		}
		if prefix != "" {
			withSlash := prefix + "/"
			if path == prefix {
				continue
			}
			if !strings.HasPrefix(path, withSlash) {
				continue
			}
			path = strings.TrimPrefix(path, withSlash)
		}
		if path == "" {
			continue
		}

		parts := strings.SplitN(path, "/", 2)
		name := parts[0]
		isDir := len(parts) == 2 || strings.HasSuffix(original, "/")
		current := items[name]
		current.name = name
		current.isDir = current.isDir || isDir
		items[name] = current
	}

	entries := make([]fileEntry, 0, len(items))
	for _, item := range items {
		entries = append(entries, fileEntry{
			Name:      item.name,
			Path:      joinArchivePrefix(prefix, item.name),
			IsDir:     item.isDir,
			IsArchive: !item.isDir && DetectFormat(strings.ToLower(item.name)) != "unknown",
			Size:      -1,
		})
	}
	sortEntries(entries)
	return entries
}

func normalizeArchivePath(path string) string {
	path = strings.ReplaceAll(strings.TrimSpace(path), "\\", "/")
	path = strings.TrimPrefix(path, "./")
	return strings.Trim(path, "/")
}

func normalizeArchivePrefix(prefix string) string {
	return normalizeArchivePath(prefix)
}

func joinArchivePrefix(prefix, name string) string {
	prefix = normalizeArchivePrefix(prefix)
	name = normalizeArchivePath(name)
	if prefix == "" {
		return name
	}
	if name == "" {
		return prefix
	}
	return prefix + "/" + name
}

func archiveParent(prefix string) string {
	prefix = normalizeArchivePrefix(prefix)
	if index := strings.LastIndex(prefix, "/"); index >= 0 {
		return prefix[:index]
	}
	return ""
}

func archiveBase(prefix string) string {
	prefix = normalizeArchivePrefix(prefix)
	if index := strings.LastIndex(prefix, "/"); index >= 0 {
		return prefix[index+1:]
	}
	return prefix
}

func isFilesystemRoot(path string) bool {
	cleaned := filepath.Clean(path)
	return filepath.Dir(cleaned) == cleaned
}
