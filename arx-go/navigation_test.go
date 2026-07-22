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
	model := initialModelAt(dir)
	model.quickSearch = "beta"
	model.panes[0].cursor = len(model.panes[0].entries) - 1
	model.searchNext()
	entry, ok := model.panes[0].selected()
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
	model := initialModelAt(root)
	if err := model.openLocation(child); err != nil {
		t.Fatal(err)
	}
	if model.panes[0].path != child {
		t.Fatalf("path=%q", model.panes[0].path)
	}
	model.historyBack()
	if model.panes[0].path != root {
		t.Fatalf("back path=%q", model.panes[0].path)
	}
	model.historyForward()
	if model.panes[0].path != child {
		t.Fatalf("forward path=%q", model.panes[0].path)
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
	model := initialModelAt(t.TempDir())
	updated, _ := model.updateBrowser(tea.KeyMsg{Type: tea.KeyF9})
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
