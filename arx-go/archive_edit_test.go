package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPreviewTextAndBinaryFiles(t *testing.T) {
	directory := t.TempDir()
	textPath := filepath.Join(directory, "note.txt")
	if err := os.WriteFile(textPath, []byte("hello\nworld"), 0o644); err != nil {
		t.Fatal(err)
	}
	lines, err := previewFile(textPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) < 2 || lines[0] != "hello" || lines[1] != "world" {
		t.Fatalf("text preview=%v", lines)
	}

	binaryPath := filepath.Join(directory, "data.bin")
	if err := os.WriteFile(binaryPath, []byte{0, 1, 2, 65}, 0o644); err != nil {
		t.Fatal(err)
	}
	lines, err = previewFile(binaryPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) == 0 || !strings.Contains(lines[0], "00 01 02 41") {
		t.Fatalf("binary preview=%v", lines)
	}
}

func TestF3OpensViewer(t *testing.T) {
	directory := t.TempDir()
	if err := os.WriteFile(filepath.Join(directory, "note.txt"), []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	m := initialModelAt(directory)
	m.panes[0].selectName("note.txt")
	updated, _ := m.Update(runeKey("f3"))
	m = updated.(model)
	if m.modal != modalViewer || len(m.viewerLines) == 0 || m.viewerLines[0] != "hello" {
		t.Fatalf("modal=%v lines=%v", m.modal, m.viewerLines)
	}
}

func TestAddToExistingArchive(t *testing.T) {
	base := t.TempDir()
	archiveDirectory := t.TempDir()
	if err := os.WriteFile(filepath.Join(base, "old.txt"), []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	if result := compressMany("tar.gz", "bundle", []string{filepath.Join(base, "old.txt")}, base, archiveDirectory, 3); result.Err != nil {
		t.Fatal(result.Err)
	}
	if err := os.WriteFile(filepath.Join(base, "new.txt"), []byte("new"), 0o644); err != nil {
		t.Fatal(err)
	}
	archive := filepath.Join(archiveDirectory, "bundle.tar.gz")
	result := addToArchive(archive, "docs", []string{filepath.Join(base, "new.txt")}, base, 3)
	if result.Err != nil {
		t.Fatal(result.Err)
	}
	items, err := readArchiveItems(archive)
	if err != nil {
		t.Fatal(err)
	}
	if !containsString(items, "old.txt") || !containsString(items, "docs/new.txt") {
		t.Fatalf("archive items=%v", items)
	}
}

func TestDeleteFromExistingArchive(t *testing.T) {
	base := t.TempDir()
	archiveDirectory := t.TempDir()
	for _, name := range []string{"keep.txt", "remove.txt"} {
		if err := os.WriteFile(filepath.Join(base, name), []byte(name), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	archive := filepath.Join(archiveDirectory, "bundle.tar.gz")
	if result := compressMany("tar.gz", "bundle", []string{filepath.Join(base, "keep.txt"), filepath.Join(base, "remove.txt")}, base, archiveDirectory, 3); result.Err != nil {
		t.Fatal(result.Err)
	}
	result := deleteFromArchive(archive, []fileEntry{{Name: "remove.txt", Path: "remove.txt"}}, 3)
	if result.Err != nil {
		t.Fatal(result.Err)
	}
	items, err := readArchiveItems(archive)
	if err != nil {
		t.Fatal(err)
	}
	if !containsString(items, "keep.txt") || containsString(items, "remove.txt") {
		t.Fatalf("archive items=%v", items)
	}
}

func TestF5AddsSelectionToArchiveInOtherPanel(t *testing.T) {
	sourceDirectory := t.TempDir()
	archiveDirectory := t.TempDir()
	if err := os.WriteFile(filepath.Join(sourceDirectory, "new.txt"), []byte("new"), 0o644); err != nil {
		t.Fatal(err)
	}
	seed := filepath.Join(archiveDirectory, "seed.txt")
	if err := os.WriteFile(seed, []byte("seed"), 0o644); err != nil {
		t.Fatal(err)
	}
	if result := compressMany("tar.gz", "bundle", []string{seed}, archiveDirectory, archiveDirectory, 3); result.Err != nil {
		t.Fatal(result.Err)
	}
	archive := filepath.Join(archiveDirectory, "bundle.tar.gz")
	right := newPane(archiveDirectory)
	right.selectName("bundle.tar.gz")
	if err := right.openSelected(); err != nil {
		t.Fatal(err)
	}

	m := initialModelAt(sourceDirectory)
	m.panes[0].selectName("new.txt")
	m.panes[1] = right
	m.active = 0
	updated, command := m.Update(runeKey("f5"))
	if command == nil {
		t.Fatal("expected add command")
	}
	m = updated.(model)
	updated, _ = m.Update(command())
	m = updated.(model)
	if m.modal == modalMessage {
		t.Fatalf("add failed: %s", m.modalMessage)
	}
	items, err := readArchiveItems(archive)
	if err != nil {
		t.Fatal(err)
	}
	if !containsString(items, "new.txt") || !containsString(items, "seed.txt") {
		t.Fatalf("archive items=%v", items)
	}
}

func TestF8InsideArchiveRequiresConfirmation(t *testing.T) {
	base := t.TempDir()
	archiveDirectory := t.TempDir()
	file := filepath.Join(base, "remove.txt")
	if err := os.WriteFile(file, []byte("remove"), 0o644); err != nil {
		t.Fatal(err)
	}
	if result := compressMany("tar.gz", "bundle", []string{file}, base, archiveDirectory, 3); result.Err != nil {
		t.Fatal(result.Err)
	}
	pane := newPane(archiveDirectory)
	pane.selectName("bundle.tar.gz")
	if err := pane.openSelected(); err != nil {
		t.Fatal(err)
	}
	pane.selectName("remove.txt")

	m := initialModelAt(archiveDirectory)
	m.panes[0] = pane
	updated, command := m.Update(runeKey("f8"))
	m = updated.(model)
	if command != nil || m.modal != modalConfirm || m.confirm != confirmArchiveDelete {
		t.Fatalf("modal=%v confirm=%v command=%v", m.modal, m.confirm, command)
	}
}
