package main

import (
	"os"
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestQuickSearchWrapsAndMatchesCaseInsensitive(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"alpha.txt", "Beta.txt", "gamma.txt"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(name), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	m := initialModelAt(dir)
	m.quickSearch = "beta"
	m.panes[0].cursor = len(m.panes[0].entries) - 1
	m.searchNext()
	entry, ok := m.panes[0].selected()
	if !ok || entry.Name != "Beta.txt" {
		t.Fatalf("selected=%q", entry.Name)
	}
}

func TestOpenLocationAndHistory(t *testing.T) {
	root := t.TempDir()
	child := filepath.Join(root, "child")
	if err := os.Mkdir(child, 0o755); err != nil {
		t.Fatal(err)
	}
	m := initialModelAt(root)
	if err := m.openLocation(child); err != nil {
		t.Fatal(err)
	}
	if m.panes[0].path != child {
		t.Fatalf("path=%q", m.panes[0].path)
	}
	m.historyBack()
	if m.panes[0].path != root {
		t.Fatalf("back path=%q", m.panes[0].path)
	}
	m.historyForward()
	if m.panes[0].path != child {
		t.Fatalf("forward path=%q", m.panes[0].path)
	}
}

func TestFavoritesPersistWithoutDuplicates(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	location := filepath.Join(t.TempDir(), "docs")
	if err := saveFavorite(location); err != nil {
		t.Fatal(err)
	}
	if err := saveFavorite(location); err != nil {
		t.Fatal(err)
	}
	items, err := loadFavorites()
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].location != location {
		t.Fatalf("favorites=%v", items)
	}
}

func TestF9OpensMenuAndCtrlHTogglesHidden(t *testing.T) {
	m := initialModelAt(t.TempDir())
	updated, _ := m.updateBrowser(tea.KeyMsg{Type: tea.KeyF9})
	menu := updated.(model)
	if menu.modal != modalNavigationMenu {
		t.Fatalf("modal=%v", menu.modal)
	}
	menu.closeModal()
	before := menu.panes[0].showHidden
	updated, _ = menu.updateBrowser(tea.KeyMsg{Type: tea.KeyCtrlH})
	toggled := updated.(model)
	if toggled.panes[0].showHidden == before {
		t.Fatal("Ctrl-H did not toggle hidden files")
	}
}

func TestF7CreatesNamedDirectory(t *testing.T) {
	directory := t.TempDir()
	m := initialModelAt(directory)
	updated, command := m.updateBrowser(tea.KeyMsg{Type: tea.KeyF7})
	if command != nil {
		t.Fatal("F7 should open a dialog without starting an operation")
	}
	dialog := updated.(model)
	if dialog.modal != modalNavigationInput || dialog.navInputKind != navigationInputMkdir {
		t.Fatalf("modal=%v input=%v", dialog.modal, dialog.navInputKind)
	}
	dialog.navInputValue = "Projects"
	updated, _ = dialog.updateModal(tea.KeyMsg{Type: tea.KeyEnter})
	dialog = updated.(model)
	if _, err := os.Stat(filepath.Join(directory, "Projects")); err != nil {
		t.Fatalf("directory was not created: %v", err)
	}
	entry, ok := dialog.panes[0].selected()
	if !ok || entry.Name != "Projects" {
		t.Fatalf("selected=%q", entry.Name)
	}
}

func TestCreateDirectoryRejectsUnsafeName(t *testing.T) {
	directory := t.TempDir()
	pane := newPane(directory)
	if err := pane.createDirectory("../escape"); err == nil {
		t.Fatal("expected path separator validation error")
	}
	if _, err := os.Stat(filepath.Join(directory, "..", "escape")); !os.IsNotExist(err) {
		t.Fatalf("unsafe directory escaped active panel: %v", err)
	}
}

func TestContextSensitiveFunctionKeyLabels(t *testing.T) {
	directory := t.TempDir()
	archive := filepath.Join(directory, "sample.zip")
	writeTestZip(t, archive)
	m := initialModelAt(directory)
	if got := m.f5Label(); got != "Copy" {
		t.Fatalf("filesystem F5=%q", got)
	}
	if got := m.f8Label(); got != "Trash" {
		t.Fatalf("filesystem F8=%q", got)
	}

	m.panes[0].selectName("sample.zip")
	if got := m.f5Label(); got != "Extract" {
		t.Fatalf("selected archive F5=%q", got)
	}

	m.panes[0].mode = paneArchive
	if got := m.f5Label(); got != "Extract" {
		t.Fatalf("archive source F5=%q", got)
	}
	if got := m.f8Label(); got != "Delete" {
		t.Fatalf("archive F8=%q", got)
	}

	m.panes[0].mode = paneFilesystem
	m.panes[1].mode = paneArchive
	if got := m.f5Label(); got != "Add" {
		t.Fatalf("archive destination F5=%q", got)
	}
}
