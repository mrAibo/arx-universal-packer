package main

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestCopyFilesystemCopiesFileDirectoryAndSymlink(t *testing.T) {
	source := t.TempDir()
	destination := t.TempDir()
	file := filepath.Join(source, "note.txt")
	if err := os.WriteFile(file, []byte("hello"), 0o640); err != nil {
		t.Fatal(err)
	}
	directory := filepath.Join(source, "docs")
	if err := os.Mkdir(directory, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, "readme.txt"), []byte("docs"), 0o644); err != nil {
		t.Fatal(err)
	}
	entries := []fileEntry{{Path: file}, {Path: directory}}

	if runtime.GOOS != "windows" {
		link := filepath.Join(source, "latest")
		if err := os.Symlink("note.txt", link); err != nil {
			t.Fatal(err)
		}
		entries = append(entries, fileEntry{Path: link})
	}

	result := copyFilesystem(entries, destination, false)
	if result.Err != nil {
		t.Fatal(result.Err)
	}
	content, err := os.ReadFile(filepath.Join(destination, "note.txt"))
	if err != nil || string(content) != "hello" {
		t.Fatalf("copied file=%q err=%v", content, err)
	}
	content, err = os.ReadFile(filepath.Join(destination, "docs", "readme.txt"))
	if err != nil || string(content) != "docs" {
		t.Fatalf("copied directory file=%q err=%v", content, err)
	}
	if runtime.GOOS != "windows" {
		target, err := os.Readlink(filepath.Join(destination, "latest"))
		if err != nil || target != "note.txt" {
			t.Fatalf("copied symlink=%q err=%v", target, err)
		}
	}
}

func TestFilesystemCopyRejectsDestinationInsideSource(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "source")
	destination := filepath.Join(source, "inside")
	if err := os.MkdirAll(destination, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := filesystemCopyConflicts([]fileEntry{{Path: source}}, destination); err == nil {
		t.Fatal("expected copy-into-self error")
	}
}

func TestF5CopyRequiresOverwriteConfirmation(t *testing.T) {
	source := t.TempDir()
	destination := t.TempDir()
	if err := os.WriteFile(filepath.Join(source, "same.txt"), []byte("new"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(destination, "same.txt"), []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	m := initialModelAt(source)
	m.panes[1] = newPane(destination)
	m.panes[0].selectName("same.txt")

	updated, command := m.startF5()
	if command != nil {
		t.Fatal("copy conflict should wait for confirmation")
	}
	confirmation := updated.(model)
	if confirmation.modal != modalConfirm || confirmation.confirm != confirmFilesystemCopy {
		t.Fatalf("modal=%v confirm=%v", confirmation.modal, confirmation.confirm)
	}

	updated, command = confirmation.updateConfirm(tea.KeyMsg{Type: tea.KeyEnter})
	if command == nil {
		t.Fatal("confirmed copy did not start")
	}
	message := command()
	updated, _ = updated.(model).Update(message)
	finished := updated.(model)
	if finished.modal != modalNone {
		t.Fatalf("unexpected modal after copy: %v", finished.modal)
	}
	content, err := os.ReadFile(filepath.Join(destination, "same.txt"))
	if err != nil || string(content) != "new" {
		t.Fatalf("overwritten file=%q err=%v", content, err)
	}
}

func TestF2OpensArchiveCreationDialog(t *testing.T) {
	source := t.TempDir()
	destination := t.TempDir()
	if err := os.WriteFile(filepath.Join(source, "note.txt"), []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	m := initialModelAt(source)
	m.panes[1] = newPane(destination)
	m.panes[0].selectName("note.txt")
	updated, command := m.updateBrowser(tea.KeyMsg{Type: tea.KeyF2})
	if command != nil {
		t.Fatal("F2 should open the archive dialog")
	}
	dialog := updated.(model)
	if dialog.modal != modalArchive || dialog.pending != actionPack {
		t.Fatalf("modal=%v pending=%v", dialog.modal, dialog.pending)
	}
}

func TestReplaceFilesystemPathKeepsDestinationWhenSourceIsMissing(t *testing.T) {
	directory := t.TempDir()
	destination := filepath.Join(directory, "existing.txt")
	if err := os.WriteFile(destination, []byte("keep"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := replaceFilesystemPath(filepath.Join(directory, "missing.txt"), destination, true); err == nil {
		t.Fatal("expected missing source error")
	}
	content, err := os.ReadFile(destination)
	if err != nil || string(content) != "keep" {
		t.Fatalf("destination changed after failed replacement: %q err=%v", content, err)
	}
}
