package main

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestMoveFilesystemRenamesFile(t *testing.T) {
	directory := t.TempDir()
	source := filepath.Join(directory, "old.txt")
	target := filepath.Join(directory, "new.txt")
	if err := os.WriteFile(source, []byte("hello"), 0o640); err != nil {
		t.Fatal(err)
	}
	result := moveFilesystem([]fileEntry{{Path: source}}, target, false)
	if result.Err != nil {
		t.Fatal(result.Err)
	}
	if _, err := os.Lstat(source); !os.IsNotExist(err) {
		t.Fatalf("source still exists: %v", err)
	}
	content, err := os.ReadFile(target)
	if err != nil || string(content) != "hello" {
		t.Fatalf("renamed content=%q err=%v", content, err)
	}
}

func TestMoveFilesystemMovesDirectoryToPanel(t *testing.T) {
	sourceRoot := t.TempDir()
	destination := t.TempDir()
	source := filepath.Join(sourceRoot, "docs")
	if err := os.Mkdir(source, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "readme.txt"), []byte("docs"), 0o644); err != nil {
		t.Fatal(err)
	}
	result := moveFilesystem([]fileEntry{{Path: source}}, destination, false)
	if result.Err != nil {
		t.Fatal(result.Err)
	}
	if _, err := os.Lstat(source); !os.IsNotExist(err) {
		t.Fatalf("source still exists: %v", err)
	}
	content, err := os.ReadFile(filepath.Join(destination, "docs", "readme.txt"))
	if err != nil || string(content) != "docs" {
		t.Fatalf("moved content=%q err=%v", content, err)
	}
}

func TestMoveRejectsDestinationInsideSource(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "source")
	destination := filepath.Join(source, "inside")
	if err := os.MkdirAll(destination, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, _, err := filesystemMovePlans([]fileEntry{{Path: source}}, destination); err == nil {
		t.Fatal("expected move-into-self error")
	}
}

func TestF6MoveRequiresOverwriteConfirmation(t *testing.T) {
	sourceRoot := t.TempDir()
	destination := t.TempDir()
	source := filepath.Join(sourceRoot, "same.txt")
	if err := os.WriteFile(source, []byte("new"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(destination, "same.txt"), []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	m := initialModelAt(sourceRoot)
	m.panes[1] = newPane(destination)
	m.panes[0].selectName("same.txt")

	updated, command := m.updateBrowser(tea.KeyMsg{Type: tea.KeyF6})
	if command != nil {
		t.Fatal("F6 should open the move target dialog")
	}
	dialog := updated.(model)
	if dialog.modal != modalNavigationInput || dialog.navInputKind != navigationInputMove {
		t.Fatalf("modal=%v input=%v", dialog.modal, dialog.navInputKind)
	}
	updated, command = dialog.updateModal(tea.KeyMsg{Type: tea.KeyEnter})
	if command != nil {
		t.Fatal("move conflict should wait for confirmation")
	}
	confirmation := updated.(model)
	if confirmation.modal != modalConfirm || confirmation.confirm != confirmFilesystemMove {
		t.Fatalf("modal=%v confirm=%v", confirmation.modal, confirmation.confirm)
	}
	updated, command = confirmation.updateConfirm(tea.KeyMsg{Type: tea.KeyEnter})
	if command == nil {
		t.Fatal("confirmed move did not start")
	}
	message := command()
	updated, _ = updated.(model).Update(message)
	finished := updated.(model)
	if finished.modal != modalNone {
		t.Fatalf("unexpected modal after move: %v %s", finished.modal, finished.modalMessage)
	}
	if _, err := os.Lstat(source); !os.IsNotExist(err) {
		t.Fatalf("source still exists: %v", err)
	}
	content, err := os.ReadFile(filepath.Join(destination, "same.txt"))
	if err != nil || string(content) != "new" {
		t.Fatalf("moved file=%q err=%v", content, err)
	}
}

func TestF6SameDirectoryDefaultsToRenameName(t *testing.T) {
	directory := t.TempDir()
	if err := os.WriteFile(filepath.Join(directory, "old.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	m := initialModelAt(directory)
	m.panes[0].selectName("old.txt")
	updated, _ := m.updateBrowser(tea.KeyMsg{Type: tea.KeyF6})
	dialog := updated.(model)
	if dialog.navInputValue != "old.txt" {
		t.Fatalf("rename default=%q", dialog.navInputValue)
	}
}

func TestMoveMultipleItemsRequiresDirectoryTarget(t *testing.T) {
	directory := t.TempDir()
	left := filepath.Join(directory, "a.txt")
	right := filepath.Join(directory, "b.txt")
	for _, path := range []string{left, right} {
		if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if _, _, err := filesystemMovePlans([]fileEntry{{Path: left}, {Path: right}}, filepath.Join(directory, "renamed")); err == nil {
		t.Fatal("expected multiple-item destination error")
	}
}

func TestMoveFilesystemCrossDeviceFallback(t *testing.T) {
	sourceRoot := t.TempDir()
	destination := t.TempDir()
	source := filepath.Join(sourceRoot, "large.dat")
	target := filepath.Join(destination, "large.dat")
	if err := os.WriteFile(source, []byte("cross-device"), 0o640); err != nil {
		t.Fatal(err)
	}
	initialReplace := func(string, string, bool) error {
		return syscall.EXDEV
	}
	if err := moveFilesystemPathWithReplace(source, target, false, initialReplace); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(source); !os.IsNotExist(err) {
		t.Fatalf("source still exists after committed fallback: %v", err)
	}
	content, err := os.ReadFile(target)
	if err != nil || string(content) != "cross-device" {
		t.Fatalf("fallback target=%q err=%v", content, err)
	}
}
