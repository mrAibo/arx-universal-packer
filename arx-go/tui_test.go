package main

import (
	"os"
	"path/filepath"
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

func TestTabSwitchesActivePanel(t *testing.T) {
	m := initialModelAt(t.TempDir())
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = updated.(model)
	if m.active != 1 {
		t.Fatalf("active=%d want 1", m.active)
	}
}

func TestEnterDirectoryAndGoBack(t *testing.T) {
	dir := t.TempDir()
	child := filepath.Join(dir, "child")
	if err := os.Mkdir(child, 0o755); err != nil {
		t.Fatal(err)
	}

	p := newPane(dir)
	p.selectName("child")
	if err := p.openSelected(); err != nil {
		t.Fatal(err)
	}
	if p.path != child {
		t.Fatalf("path=%q want %q", p.path, child)
	}
	if err := p.goUp(); err != nil {
		t.Fatal(err)
	}
	if p.path != dir {
		t.Fatalf("path=%q want %q", p.path, dir)
	}
	selected, ok := p.selected()
	if !ok || selected.Name != "child" {
		t.Fatalf("selected=%+v ok=%v", selected, ok)
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

func TestArchivePanelNavigatesVirtualDirectories(t *testing.T) {
	p := pane{
		mode:         paneArchive,
		archivePath:  "/tmp/test.tar.gz",
		archiveItems: []string{"docs/readme.txt", "docs/manual/a.txt", "root.txt"},
	}
	if err := p.loadArchiveView(); err != nil {
		t.Fatal(err)
	}
	p.selectName("docs")
	if err := p.openSelected(); err != nil {
		t.Fatal(err)
	}
	if p.archivePrefix != "docs" {
		t.Fatalf("prefix=%q want docs", p.archivePrefix)
	}
	if !containsEntry(p.entries, "readme.txt") {
		t.Fatalf("entries=%+v", p.entries)
	}
	if err := p.goUp(); err != nil {
		t.Fatal(err)
	}
	if p.archivePrefix != "" {
		t.Fatalf("prefix=%q want empty", p.archivePrefix)
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

func TestF5OnRegularFileOpensFormatDialog(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "data.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	m := initialModelAt(dir)
	m.panes[0].selectName("data.txt")
	updated, _ := m.Update(runeKey("f5"))
	m = updated.(model)
	if m.modal != modalFormat || m.pending != actionPack {
		t.Fatalf("modal=%v pending=%v", m.modal, m.pending)
	}
	if m.pendingPath != filepath.Join(dir, "data.txt") {
		t.Fatalf("pendingPath=%q", m.pendingPath)
	}
}

func TestF6RejectsNonArchive(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "data.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	m := initialModelAt(dir)
	m.panes[0].selectName("data.txt")
	updated, _ := m.Update(runeKey("f6"))
	m = updated.(model)
	if m.modal != modalMessage {
		t.Fatalf("modal=%v want modalMessage", m.modal)
	}
}

func TestAvailableArchiveNameAvoidsOverwrite(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "backup.tar.zst"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := availableArchiveName(dir, "backup", "tar.zst"); got != "backup-1" {
		t.Fatalf("got=%q want backup-1", got)
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
