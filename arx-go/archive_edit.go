package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"
)

const (
	previewReadLimit   = 2 << 20
	binaryPreviewLimit = 4 << 10
)

type confirmKind int

const (
	confirmNone confirmKind = iota
	confirmArchiveDelete
	confirmFilesystemCopy
)

func (m model) startViewer() (tea.Model, tea.Cmd) {
	pane := m.panes[m.active]
	entry, ok := pane.selected()
	if !ok || entry.Name == ".." {
		m.showError(fmt.Errorf("select a file to view"))
		return m, nil
	}
	if entry.IsDir {
		m.showError(fmt.Errorf("directories cannot be viewed"))
		return m, nil
	}

	title := entry.Name
	var lines []string
	var err error
	switch {
	case pane.mode == paneArchive:
		title = filepath.Base(pane.archivePath) + ":/" + entry.Path
		lines, err = previewArchiveMember(pane.archivePath, entry.Path)
	case entry.IsArchive:
		var items []string
		items, err = readArchiveItems(entry.Path)
		if err == nil {
			lines = append([]string{fmt.Sprintf("Archive: %s", entry.Path), fmt.Sprintf("Entries: %d", len(items)), ""}, items...)
		}
	default:
		lines, err = previewFile(entry.Path)
	}
	if err != nil {
		m.showError(err)
		return m, nil
	}
	if len(lines) == 0 {
		lines = []string{"(empty file)"}
	}
	m.modal = modalViewer
	m.modalTitle = title
	m.viewerLines = lines
	m.viewerOffset = 0
	m.viewerColumn = 0
	return m, nil
}

func previewArchiveMember(archivePath, member string) ([]string, error) {
	member = normalizeArchivePath(member)
	if member == "" {
		return nil, fmt.Errorf("unsafe archive member path")
	}
	temporary, err := os.MkdirTemp("", "arx-view-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(temporary)

	result := extractSelected(archivePath, []string{member}, temporary)
	if result.Err != nil {
		return nil, result.Err
	}
	return previewFile(filepath.Join(temporary, filepath.FromSlash(member)))
}

func previewFile(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	data, err := io.ReadAll(io.LimitReader(file, previewReadLimit+1))
	if err != nil {
		return nil, err
	}
	truncated := len(data) > previewReadLimit
	if truncated {
		data = data[:previewReadLimit]
	}

	if bytes.IndexByte(data, 0) >= 0 || !utf8.Valid(data) {
		if len(data) > binaryPreviewLimit {
			data = data[:binaryPreviewLimit]
			truncated = true
		}
		lines := hexPreview(data)
		if truncated {
			lines = append(lines, "", "[binary preview truncated]")
		}
		return lines, nil
	}

	text := strings.ReplaceAll(string(data), "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	text = strings.ReplaceAll(text, "\t", "    ")
	lines := strings.Split(text, "\n")
	if truncated {
		lines = append(lines, "", "[text preview truncated at 2 MiB]")
	}
	return lines, nil
}

func hexPreview(data []byte) []string {
	lines := make([]string, 0, (len(data)+15)/16)
	for offset := 0; offset < len(data); offset += 16 {
		end := offset + 16
		if end > len(data) {
			end = len(data)
		}
		chunk := data[offset:end]
		var hexPart strings.Builder
		var textPart strings.Builder
		for i := 0; i < 16; i++ {
			if i < len(chunk) {
				fmt.Fprintf(&hexPart, "%02x ", chunk[i])
				if chunk[i] >= 32 && chunk[i] < 127 {
					textPart.WriteByte(chunk[i])
				} else {
					textPart.WriteByte('.')
				}
			} else {
				hexPart.WriteString("   ")
			}
		}
		lines = append(lines, fmt.Sprintf("%08x  %s |%s|", offset, hexPart.String(), textPart.String()))
	}
	return lines
}

func (m model) updateViewer(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	page := m.viewerRows()
	maxOffset := len(m.viewerLines) - page
	if maxOffset < 0 {
		maxOffset = 0
	}

	switch msg.String() {
	case "esc", "f3", "q", "enter":
		m.closeModal()
	case "up", "k":
		m.viewerOffset--
	case "down", "j":
		m.viewerOffset++
	case "pgup":
		m.viewerOffset -= page
	case "pgdown", "space":
		m.viewerOffset += page
	case "home", "g":
		m.viewerOffset = 0
	case "end", "G":
		m.viewerOffset = maxOffset
	case "left":
		m.viewerColumn -= 4
	case "right":
		m.viewerColumn += 4
	}
	if m.viewerOffset < 0 {
		m.viewerOffset = 0
	}
	if m.viewerOffset > maxOffset {
		m.viewerOffset = maxOffset
	}
	if m.viewerColumn < 0 {
		m.viewerColumn = 0
	}
	return m, nil
}

func (m model) updateViewerMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	switch msg.Button {
	case tea.MouseButtonWheelUp:
		m.viewerOffset -= 3
	case tea.MouseButtonWheelDown:
		m.viewerOffset += 3
	default:
		return m, nil
	}
	if m.viewerOffset < 0 {
		m.viewerOffset = 0
	}
	maxOffset := len(m.viewerLines) - m.viewerRows()
	if maxOffset < 0 {
		maxOffset = 0
	}
	if m.viewerOffset > maxOffset {
		m.viewerOffset = maxOffset
	}
	return m, nil
}

func (m model) viewerRows() int {
	rows := m.height - 8
	if rows < 5 {
		return 5
	}
	return rows
}

func (m model) startArchiveDelete() (tea.Model, tea.Cmd) {
	pane := m.panes[m.active]
	if pane.mode != paneArchive {
		pane.clearMarks()
		m.panes[m.active] = pane
		m.status = "Marks cleared"
		return m, nil
	}
	entries, err := pane.operationEntries()
	if err != nil {
		m.showError(err)
		return m, nil
	}
	m.modal = modalConfirm
	m.modalTitle = "Delete from archive"
	m.modalMessage = fmt.Sprintf("Delete %d selected item(s) from %s?\n\nThe archive is rebuilt safely before the original is replaced.", len(entries), filepath.Base(pane.archivePath))
	m.confirm = confirmArchiveDelete
	m.confirmArchive = pane.archivePath
	m.confirmEntries = append([]fileEntry(nil), entries...)
	return m, nil
}

func (m model) updateConfirm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "n", "q":
		m.closeModal()
		return m, nil
	case "enter", "y", "f5", "f8":
		confirmation := m.confirm
		archive := m.confirmArchive
		entries := append([]fileEntry(nil), m.confirmEntries...)
		destination := m.confirmDestination
		m.closeModal()
		switch confirmation {
		case confirmArchiveDelete:
			return m.startOperation(fmt.Sprintf("Deleting %d archive item(s)...", len(entries)), func() Result {
				return deleteFromArchive(archive, entries, defaultLevel)
			})
		case confirmFilesystemCopy:
			return m.runFilesystemCopy(entries, destination, true)
		default:
			return m, nil
		}
	default:
		return m, nil
	}
}

func (p *pane) refresh() error {
	if p.mode != paneArchive {
		return p.reload()
	}
	items, err := readArchiveItems(p.archivePath)
	if err != nil {
		p.err = err.Error()
		return err
	}
	p.archiveItems = items
	return p.loadArchiveView()
}

func addToArchive(archivePath, prefix string, sources []string, baseDir string, level int) Result {
	return rebuildArchive(archivePath, level, func(staging string) error {
		destination := staging
		if prefix = normalizeArchivePrefix(prefix); prefix != "" {
			destination = filepath.Join(staging, filepath.FromSlash(prefix))
		}
		if err := os.MkdirAll(destination, 0o755); err != nil {
			return err
		}
		relative, err := relativeSources(baseDir, sources)
		if err != nil {
			return err
		}
		for _, source := range relative {
			absSource := filepath.Join(baseDir, filepath.FromSlash(source))
			name := filepath.Base(filepath.FromSlash(source))
			target := filepath.Join(destination, name)
			if _, err := os.Lstat(target); err == nil {
				return fmt.Errorf("archive already contains %s", filepath.ToSlash(filepath.Join(prefix, name)))
			} else if !os.IsNotExist(err) {
				return err
			}
			if err := copyPath(absSource, target); err != nil {
				return err
			}
		}
		return nil
	})
}

func deleteFromArchive(archivePath string, entries []fileEntry, level int) Result {
	return rebuildArchive(archivePath, level, func(staging string) error {
		for _, entry := range entries {
			member := normalizeArchivePath(entry.Path)
			if member == "" {
				return fmt.Errorf("unsafe archive member path: %s", entry.Path)
			}
			target := filepath.Join(staging, filepath.FromSlash(member))
			if _, err := os.Lstat(target); err != nil {
				return fmt.Errorf("archive member not found: %s", member)
			}
			if err := os.RemoveAll(target); err != nil {
				return err
			}
		}
		return nil
	})
}

func rebuildArchive(archivePath string, level int, mutate func(string) error) Result {
	archive, err := filepath.Abs(archivePath)
	if err != nil {
		return Result{Err: err}
	}
	info, err := os.Stat(archive)
	if err != nil {
		return Result{Err: err}
	}
	format := DetectArchiveFormat(archive)
	if format == "unknown" {
		return Result{Err: fmt.Errorf("unsupported archive format: %s", archivePath)}
	}

	staging, err := os.MkdirTemp("", "arx-edit-*")
	if err != nil {
		return Result{Err: err}
	}
	defer os.RemoveAll(staging)
	if extracted := extract(archive, staging); extracted.Err != nil {
		return extracted
	}
	if err := rejectSymlinks(staging); err != nil {
		return Result{Err: err}
	}
	if err := mutate(staging); err != nil {
		return Result{Err: err}
	}

	sources, err := topLevelSources(staging)
	if err != nil {
		return Result{Err: err}
	}
	if len(sources) == 0 {
		return Result{Err: fmt.Errorf("cannot leave an empty archive")}
	}

	outputDir, err := os.MkdirTemp(filepath.Dir(archive), ".arx-rebuild-*")
	if err != nil {
		return Result{Err: err}
	}
	defer os.RemoveAll(outputDir)
	name := "replacement"
	created := compressMany(format, name, sources, staging, outputDir, level)
	if created.Err != nil {
		return created
	}
	replacement := filepath.Join(outputDir, name+"."+format)
	if tested := testArchive(replacement); tested.Err != nil {
		return tested
	}
	if err := os.Chmod(replacement, info.Mode().Perm()); err != nil {
		return Result{Err: err}
	}
	if err := os.Rename(replacement, archive); err != nil {
		return Result{Err: err}
	}
	return Result{Output: "Archive updated: " + archive}
}

func topLevelSources(directory string) ([]string, error) {
	items, err := os.ReadDir(directory)
	if err != nil {
		return nil, err
	}
	sources := make([]string, 0, len(items))
	for _, item := range items {
		sources = append(sources, filepath.Join(directory, item.Name()))
	}
	return sources, nil
}

func copyPath(source, destination string) error {
	info, err := os.Lstat(source)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		// ponytail: archive editing rejects symlinks until link-target validation is implemented.
		return fmt.Errorf("symbolic links are not supported while editing archives: %s", source)
	}
	if info.IsDir() {
		if err := os.Mkdir(destination, info.Mode().Perm()); err != nil {
			return err
		}
		items, err := os.ReadDir(source)
		if err != nil {
			return err
		}
		for _, item := range items {
			if err := copyPath(filepath.Join(source, item.Name()), filepath.Join(destination, item.Name())); err != nil {
				return err
			}
		}
		return nil
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("unsupported file type: %s", source)
	}

	input, err := os.Open(source)
	if err != nil {
		return err
	}
	defer input.Close()
	output, err := os.OpenFile(destination, os.O_CREATE|os.O_EXCL|os.O_WRONLY, info.Mode().Perm())
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(output, input)
	closeErr := output.Close()
	if copyErr != nil {
		return copyErr
	}
	return closeErr
}

func rejectSymlinks(root string) error {
	return filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("archive editing is disabled for archives containing symbolic links: %s", path)
		}
		return nil
	})
}
