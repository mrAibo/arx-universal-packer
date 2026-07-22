package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRestoreLastTrashRoundTrip(t *testing.T) {
	dataHome := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dataHome)
	root := t.TempDir()
	original := filepath.Join(root, "note.txt")
	if err := os.WriteFile(original, []byte("hello"), 0o640); err != nil {
		t.Fatal(err)
	}
	filesDir, infoDir, err := trashDirectories()
	if err != nil {
		t.Fatal(err)
	}
	record, err := trashFilesystemPathRecord(original, filesDir, infoDir, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	setLastTrashRecords([]trashRecord{record})
	result := restoreTrashRecords(lastTrashRecords())
	if result.Err != nil {
		t.Fatal(result.Err)
	}
	content, err := os.ReadFile(original)
	if err != nil || string(content) != "hello" {
		t.Fatalf("restored content=%q err=%v", content, err)
	}
	if _, err := os.Lstat(record.dataPath); !os.IsNotExist(err) {
		t.Fatalf("trash data still exists: %v", err)
	}
	if _, err := os.Lstat(record.infoPath); !os.IsNotExist(err) {
		t.Fatalf("trash metadata still exists: %v", err)
	}
	if len(lastTrashRecords()) != 0 {
		t.Fatal("undo batch was not cleared")
	}
}

func TestRestoreRefusesExistingOriginal(t *testing.T) {
	root := t.TempDir()
	original := filepath.Join(root, "same.txt")
	data := filepath.Join(root, "trash-data")
	info := filepath.Join(root, "trash-info")
	if err := os.WriteFile(original, []byte("current"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(data, []byte("deleted"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(info, []byte("metadata"), 0o600); err != nil {
		t.Fatal(err)
	}
	err := restoreTrashRecord(trashRecord{originalPath: original, dataPath: data, infoPath: info})
	if err == nil {
		t.Fatal("expected existing-path conflict")
	}
	content, _ := os.ReadFile(original)
	if string(content) != "current" {
		t.Fatalf("original was overwritten: %q", content)
	}
	if _, err := os.Lstat(data); err != nil {
		t.Fatalf("trash data was removed: %v", err)
	}
}

func TestRestoreKeepsRemainingBatchAfterFailure(t *testing.T) {
	root := t.TempDir()
	first := trashRecord{
		originalPath: filepath.Join(root, "first.txt"),
		dataPath:     filepath.Join(root, "first-trash"),
		infoPath:     filepath.Join(root, "first.trashinfo"),
	}
	second := trashRecord{
		originalPath: filepath.Join(root, "second.txt"),
		dataPath:     filepath.Join(root, "missing-trash"),
		infoPath:     filepath.Join(root, "second.trashinfo"),
	}
	if err := os.WriteFile(first.dataPath, []byte("first"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(first.infoPath, []byte("metadata"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(second.infoPath, []byte("metadata"), 0o600); err != nil {
		t.Fatal(err)
	}
	setLastTrashRecords([]trashRecord{first, second})
	result := restoreTrashRecords(lastTrashRecords())
	if result.Err == nil {
		t.Fatal("expected partial restore failure")
	}
	if _, err := os.Stat(first.originalPath); err != nil {
		t.Fatalf("first item was not restored: %v", err)
	}
	remaining := lastTrashRecords()
	if len(remaining) != 1 || remaining[0].originalPath != second.originalPath {
		t.Fatalf("remaining batch=%+v", remaining)
	}
}
