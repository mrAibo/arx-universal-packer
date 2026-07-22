package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func runeKey(value string) tea.Msg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(value)}
}

func TestInitialModelCreatesTwoFilesystemPanels(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "sample.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	m := initialModelAt(dir)
	for i, panel := range m.panes {
		if panel.mode != paneFilesystem {
			t.Fatalf("panel %d mode=%v", i, panel.mode)
		}
		if panel.path != dir {
			t.Fatalf("panel %d path=%q want %q", i, panel.path, dir)
		}
		if !containsEntry(panel.entries, "sample.txt") {
			t.Fatalf("panel %d does not contain sample.txt", i)
		}
	}
}

func TestSpaceMarksMultipleItemsAndAdvances(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"a.txt", "b.txt", "c.txt"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(name), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	m := initialModelAt(dir)
	m.panes[0].selectName("a.txt")

	updated, _ := m.Update(runeKey(" "))
	m = updated.(model)
	updated, _ = m.Update(runeKey(" "))
	m = updated.(model)

	marked := m.panes[0].markedEntries()
	if len(marked) != 2 || marked[0].Name != "a.txt" || marked[1].Name != "b.txt" {
		t.Fatalf("marked=%+v", marked)
	}
	if !strings.Contains(m.status, "2 marked") {
		t.Fatalf("status=%q", m.status)
	}
}

func TestCtrlAMarksAllAndCtrlUClears(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"a.txt", "b.txt"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	m := initialModelAt(dir)
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlA})
	m = updated.(model)
	if got := len(m.panes[0].markedEntries()); got != 2 {
		t.Fatalf("marked=%d want 2", got)
	}
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlU})
	m = updated.(model)
	if got := len(m.panes[0].markedEntries()); got != 0 {
		t.Fatalf("marked=%d want 0", got)
	}
}

func TestF2WithMarkedItemsOpensNamedArchiveDialog(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"a.txt", "b.txt"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	m := initialModelAt(dir)
	m.panes[0].markAll()
	updated, _ := m.Update(runeKey("f2"))
	m = updated.(model)

	if m.modal != modalArchive || m.pending != actionPack {
		t.Fatalf("modal=%v pending=%v", m.modal, m.pending)
	}
	if len(m.pendingSources) != 2 {
		t.Fatalf("pending sources=%v", m.pendingSources)
	}
	if m.archiveName != "archive" {
		t.Fatalf("archiveName=%q want archive", m.archiveName)
	}
	if m.panes[1].path != dir {
		t.Fatalf("destination panel path=%q", m.panes[1].path)
	}
}

func TestArchiveDialogNameCanBeReplaced(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "data.txt")
	if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	m := initialModelAt(dir)
	m.panes[0].selectName("data.txt")
	updated, _ := m.Update(runeKey("f2"))
	m = updated.(model)
	if !m.nameReplaceMode {
		t.Fatal("dialog should initially select the proposed name")
	}
	updated, _ = m.Update(runeKey("backup-2026"))
	m = updated.(model)
	if m.archiveName != "backup-2026" {
		t.Fatalf("archiveName=%q", m.archiveName)
	}
}

func TestNormalizeArchiveNameStripsKnownExtension(t *testing.T) {
	name, err := normalizeArchiveName(" backup.tar.gz ", "zip")
	if err != nil {
		t.Fatal(err)
	}
	if name != "backup" {
		t.Fatalf("name=%q want backup", name)
	}
	if _, err := normalizeArchiveName("../backup", "zip"); err == nil {
		t.Fatal("expected path separator validation error")
	}
}

func TestCompressManyCreatesArchiveWithMultipleSources(t *testing.T) {
	base := t.TempDir()
	target := t.TempDir()
	if err := os.WriteFile(filepath.Join(base, "a.txt"), []byte("a"), 0o644); err != nil {
		t.Fatal(err)
	}
	folder := filepath.Join(base, "docs")
	if err := os.Mkdir(folder, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(folder, "b.txt"), []byte("b"), 0o644); err != nil {
		t.Fatal(err)
	}

	result := compressMany("tar.gz", "bundle", []string{filepath.Join(base, "a.txt"), folder}, base, target, 3)
	if result.Err != nil {
		t.Fatal(result.Err)
	}
	items, err := readArchiveItems(filepath.Join(target, "bundle.tar.gz"))
	if err != nil {
		t.Fatal(err)
	}
	if !containsString(items, "a.txt") || !containsString(items, "docs/b.txt") {
		t.Fatalf("archive items=%v", items)
	}
}

func TestSelectiveExtractionFromOpenedArchive(t *testing.T) {
	base := t.TempDir()
	archiveDir := t.TempDir()
	target := t.TempDir()
	docs := filepath.Join(base, "docs")
	if err := os.Mkdir(docs, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(docs, "a.txt"), []byte("a"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(base, "root.txt"), []byte("root"), 0o644); err != nil {
		t.Fatal(err)
	}
	archiveResult := compressMany("tar.gz", "sample", []string{docs, filepath.Join(base, "root.txt")}, base, archiveDir, 3)
	if archiveResult.Err != nil {
		t.Fatal(archiveResult.Err)
	}

	archivePath := filepath.Join(archiveDir, "sample.tar.gz")
	left := newPane(archiveDir)
	left.selectName("sample.tar.gz")
	if err := left.openSelected(); err != nil {
		t.Fatal(err)
	}
	left.selectName("docs")
	left.toggleMarkIndex(left.cursor, 10, false)

	m := initialModelAt(archiveDir)
	m.panes[0] = left
	m.panes[1] = newPane(target)
	m.active = 0
	updated, cmd := m.Update(runeKey("f5"))
	if cmd == nil {
		t.Fatal("expected extraction command")
	}
	m = updated.(model)
	message := cmd()
	updated, _ = m.Update(message)
	m = updated.(model)
	if m.modal == modalMessage {
		t.Fatalf("extraction failed: %s", m.modalMessage)
	}
	if _, err := os.Stat(filepath.Join(target, "docs", "a.txt")); err != nil {
		t.Fatalf("selected directory not extracted: %v", err)
	}
	if _, err := os.Stat(filepath.Join(target, "root.txt")); !os.IsNotExist(err) {
		t.Fatalf("unselected root.txt should not be extracted, err=%v", err)
	}
	if left.archivePath != archivePath {
		t.Fatalf("archive path changed unexpectedly: %q", left.archivePath)
	}
}

func TestMouseClickSelectsAndRightClickMarks(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("a"), 0o644); err != nil {
		t.Fatal(err)
	}
	m := initialModelAt(dir)
	m.width = 100
	m.height = 30

	// The temp directory has a parent entry at row 4, so a.txt is row 5.
	updated, _ := m.Update(tea.MouseMsg{X: 2, Y: 5, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft})
	m = updated.(model)
	entry, ok := m.panes[0].selected()
	if !ok || entry.Name != "a.txt" {
		t.Fatalf("selected=%+v ok=%v", entry, ok)
	}
	updated, _ = m.Update(tea.MouseMsg{X: 2, Y: 5, Action: tea.MouseActionPress, Button: tea.MouseButtonRight})
	m = updated.(model)
	if got := len(m.panes[0].markedEntries()); got != 1 {
		t.Fatalf("marked=%d want 1", got)
	}
}

func TestMouseDoubleClickOpensDirectory(t *testing.T) {
	dir := t.TempDir()
	child := filepath.Join(dir, "child")
	if err := os.Mkdir(child, 0o755); err != nil {
		t.Fatal(err)
	}
	m := initialModelAt(dir)
	m.width = 100
	m.height = 30
	// child is the first real entry after '..'.
	click := tea.MouseMsg{X: 2, Y: 5, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft}
	updated, _ := m.Update(click)
	m = updated.(model)
	updated, _ = m.Update(click)
	m = updated.(model)
	if m.panes[0].path != child {
		t.Fatalf("path=%q want %q", m.panes[0].path, child)
	}
}

func TestKeyBarUsesFunctionKeyNames(t *testing.T) {
	m := initialModelAt(t.TempDir())
	m.width = 120
	view := m.View()
	if !strings.Contains(view, "F1") || !strings.Contains(view, "F10") {
		t.Fatalf("function key labels missing from view: %q", view)
	}
}

func TestArchivePathNormalizationRejectsTraversal(t *testing.T) {
	for _, value := range []string{"../escape", "/etc/passwd", "C:/Windows/file", "a/../../escape"} {
		if got := normalizeArchivePath(value); got != "" {
			t.Fatalf("normalizeArchivePath(%q)=%q want empty", value, got)
		}
	}
	if got := normalizeArchivePath("./docs/readme.txt"); got != "docs/readme.txt" {
		t.Fatalf("got=%q", got)
	}
}

func TestArchiveHierarchyBuildsImmediateChildren(t *testing.T) {
	paths := []string{
		"docs/readme.txt",
		"docs/manual/intro.md",
		"bin/tool",
		"root.txt",
	}
	root := buildArchiveEntries(paths, "")
	if !entryIsDirectory(root, "docs") || !entryIsDirectory(root, "bin") {
		t.Fatalf("root entries=%+v", root)
	}
	if !containsEntry(root, "root.txt") {
		t.Fatalf("root entries=%+v", root)
	}

	docs := buildArchiveEntries(paths, "docs")
	if !containsEntry(docs, "readme.txt") || !entryIsDirectory(docs, "manual") {
		t.Fatalf("docs entries=%+v", docs)
	}
}

func TestHiddenFilesToggle(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".secret"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	p := newPane(dir)
	if containsEntry(p.entries, ".secret") {
		t.Fatal("hidden file shown by default")
	}
	p.showHidden = true
	if err := p.reload(); err != nil {
		t.Fatal(err)
	}
	if !containsEntry(p.entries, ".secret") {
		t.Fatal("hidden file not shown after toggle")
	}
}

func TestParse7zPathsSkipsArchiveHeader(t *testing.T) {
	output := "Path = backup.7z\nType = 7z\n\nPath = docs/readme.txt\nAttributes = A\n\nPath = empty\nAttributes = D\n\nPath = bin/tool\nAttributes = A\n"
	got := parse7zPaths(output, "/tmp/backup.7z")
	if len(got) != 3 || got[0] != "docs/readme.txt" || got[1] != "empty/" || got[2] != "bin/tool" {
		t.Fatalf("got=%v", got)
	}
}

func containsEntry(entries []fileEntry, name string) bool {
	for _, entry := range entries {
		if entry.Name == name {
			return true
		}
	}
	return false
}

func entryIsDirectory(entries []fileEntry, name string) bool {
	for _, entry := range entries {
		if entry.Name == name {
			return entry.IsDir
		}
	}
	return false
}

func containsString(values []string, wanted string) bool {
	for _, value := range values {
		if normalizeArchivePath(value) == wanted {
			return true
		}
	}
	return false
}
