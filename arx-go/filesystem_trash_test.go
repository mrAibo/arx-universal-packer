package main

import (
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func TestTrashFilesystemMovesFileAndWritesMetadata(t *testing.T) {
	dataHome := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dataHome)
	sourceDir := t.TempDir()
	source := filepath.Join(sourceDir, "note.txt")
	if err := os.WriteFile(source, []byte("hello"), 0o640); err != nil {
		t.Fatal(err)
	}
	result := trashFilesystem([]fileEntry{{Path: source}})
	if result.Err != nil {
		t.Fatal(result.Err)
	}
	if _, err := os.Lstat(source); !os.IsNotExist(err) {
		t.Fatalf("source still exists: %v", err)
	}
	trashed := filepath.Join(dataHome, "Trash", "files", "note.txt")
	content, err := os.ReadFile(trashed)
	if err != nil || string(content) != "hello" {
		t.Fatalf("trashed content=%q err=%v", content, err)
	}
	metadata, err := os.ReadFile(filepath.Join(dataHome, "Trash", "info", "note.txt.trashinfo"))
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(string(metadata), "\n")
	if len(lines) < 3 || lines[0] != "[Trash Info]" || !strings.HasPrefix(lines[1], "Path=") || !strings.HasPrefix(lines[2], "DeletionDate=") {
		t.Fatalf("metadata=%q", metadata)
	}
	decoded, err := url.PathUnescape(strings.TrimPrefix(lines[1], "Path="))
	if err != nil || decoded != source {
		t.Fatalf("metadata path=%q err=%v", decoded, err)
	}
}

func TestTrashFilesystemUsesCollisionSafeName(t *testing.T) {
	dataHome := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dataHome)
	filesDir, infoDir, err := trashDirectories()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(filesDir, "same.txt"), []byte("old"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(infoDir, "same.txt.trashinfo"), []byte("old"), 0o600); err != nil {
		t.Fatal(err)
	}
	sourceDir := t.TempDir()
	source := filepath.Join(sourceDir, "same.txt")
	if err := os.WriteFile(source, []byte("new"), 0o600); err != nil {
		t.Fatal(err)
	}
	if result := trashFilesystem([]fileEntry{{Path: source}}); result.Err != nil {
		t.Fatal(result.Err)
	}
	content, err := os.ReadFile(filepath.Join(filesDir, "same.txt.1"))
	if err != nil || string(content) != "new" {
		t.Fatalf("collision content=%q err=%v", content, err)
	}
}

func TestTrashFilesystemPreservesSymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink semantics differ on Windows")
	}
	dataHome := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dataHome)
	sourceDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(sourceDir, "target.txt"), []byte("target"), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(sourceDir, "link")
	if err := os.Symlink("target.txt", link); err != nil {
		t.Fatal(err)
	}
	if result := trashFilesystem([]fileEntry{{Path: link}}); result.Err != nil {
		t.Fatal(result.Err)
	}
	trashed := filepath.Join(dataHome, "Trash", "files", "link")
	target, err := os.Readlink(trashed)
	if err != nil || target != "target.txt" {
		t.Fatalf("trashed symlink=%q err=%v", target, err)
	}
	if _, err := os.Stat(filepath.Join(sourceDir, "target.txt")); err != nil {
		t.Fatalf("symlink target was affected: %v", err)
	}
}

func TestFilesystemTrashSummaryDoesNotFollowSymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink semantics differ on Windows")
	}
	root := t.TempDir()
	target := filepath.Join(root, "target")
	if err := os.Mkdir(target, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(target, "large"), make([]byte, 1024), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "link")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	files, directories, bytes, err := filesystemTrashSummary([]fileEntry{{Path: link}})
	if err != nil {
		t.Fatal(err)
	}
	if files != 1 || directories != 0 || bytes >= 1024 {
		t.Fatalf("summary files=%d dirs=%d bytes=%d", files, directories, bytes)
	}
}

func TestF8OpensTrashConfirmation(t *testing.T) {
	directory := t.TempDir()
	if err := os.WriteFile(filepath.Join(directory, "note.txt"), []byte("hello"), 0o600); err != nil {
		t.Fatal(err)
	}
	m := initialModelAt(directory)
	m.panes[0].selectName("note.txt")
	updated, command := m.updateBrowser(tea.KeyMsg{Type: tea.KeyF8})
	if command != nil {
		t.Fatal("F8 should wait for confirmation")
	}
	confirmation := updated.(model)
	if confirmation.modal != modalConfirm || confirmation.confirm != confirmFilesystemTrash {
		t.Fatalf("modal=%v confirm=%v", confirmation.modal, confirmation.confirm)
	}
	if !strings.Contains(confirmation.modalMessage, "restored") {
		t.Fatalf("confirmation=%q", confirmation.modalMessage)
	}
}

func TestTrashInfoUsesProvidedDeletionTime(t *testing.T) {
	dataHome := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dataHome)
	filesDir, infoDir, err := trashDirectories()
	if err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(t.TempDir(), "dated.txt")
	if err := os.WriteFile(source, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	deletedAt := time.Date(2026, 7, 22, 14, 30, 15, 0, time.Local)
	if err := trashFilesystemPath(source, filesDir, infoDir, deletedAt); err != nil {
		t.Fatal(err)
	}
	metadata, err := os.ReadFile(filepath.Join(infoDir, "dated.txt.trashinfo"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(metadata), "DeletionDate=2026-07-22T14:30:15") {
		t.Fatalf("metadata=%q", metadata)
	}
}
