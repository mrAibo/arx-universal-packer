package main

import (
	"os"
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestCopyConflictReplaceAndSkip(t *testing.T) {
	source := t.TempDir()
	destination := t.TempDir()
	for _, name := range []string{"a.txt", "b.txt"} {
		if err := os.WriteFile(filepath.Join(source, name), []byte("new-"+name), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(destination, name), []byte("old-"+name), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	entries := []fileEntry{{Path: filepath.Join(source, "a.txt")}, {Path: filepath.Join(source, "b.txt")}}
	plans, err := filesystemCopyPlans(entries, destination)
	if err != nil {
		t.Fatal(err)
	}
	if err := applyCopyDecision(plans, 0, copyConflictReplace, ""); err != nil {
		t.Fatal(err)
	}
	if err := applyCopyDecision(plans, 1, copyConflictSkip, ""); err != nil {
		t.Fatal(err)
	}
	result := copyFilesystemPlans(plans)
	if result.Err != nil {
		t.Fatal(result.Err)
	}
	content, _ := os.ReadFile(filepath.Join(destination, "a.txt"))
	if string(content) != "new-a.txt" {
		t.Fatalf("a.txt=%q", content)
	}
	content, _ = os.ReadFile(filepath.Join(destination, "b.txt"))
	if string(content) != "old-b.txt" {
		t.Fatalf("b.txt=%q", content)
	}
}

func TestCopyConflictRenameDoesNotOverwrite(t *testing.T) {
	source := t.TempDir()
	destination := t.TempDir()
	if err := os.WriteFile(filepath.Join(source, "note.txt"), []byte("new"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(destination, "note.txt"), []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	plans, err := filesystemCopyPlans([]fileEntry{{Path: filepath.Join(source, "note.txt")}}, destination)
	if err != nil {
		t.Fatal(err)
	}
	if err := applyCopyDecision(plans, 0, copyConflictRename, "note-copy.txt"); err != nil {
		t.Fatal(err)
	}
	result := copyFilesystemPlans(plans)
	if result.Err != nil {
		t.Fatal(result.Err)
	}
	original, _ := os.ReadFile(filepath.Join(destination, "note.txt"))
	copyContent, _ := os.ReadFile(filepath.Join(destination, "note-copy.txt"))
	if string(original) != "old" || string(copyContent) != "new" {
		t.Fatalf("original=%q copy=%q", original, copyContent)
	}
}

func TestCopyConflictRejectsUnsafeRename(t *testing.T) {
	plans := []filesystemCopyPlan{{target: filepath.Join(t.TempDir(), "a.txt"), conflict: true}}
	for _, value := range []string{"", ".", "..", "../escape", "a/b"} {
		if err := applyCopyDecision(plans, 0, copyConflictRename, value); err == nil {
			t.Fatalf("rename %q should fail", value)
		}
	}
}

func TestCopyConflictDialogApplyAllSkip(t *testing.T) {
	source := t.TempDir()
	destination := t.TempDir()
	entries := make([]fileEntry, 0, 2)
	for _, name := range []string{"a.txt", "b.txt"} {
		path := filepath.Join(source, name)
		if err := os.WriteFile(path, []byte("new"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(destination, name), []byte("old"), 0o644); err != nil {
			t.Fatal(err)
		}
		entries = append(entries, fileEntry{Path: path})
	}
	m := initialModelAt(source)
	updated, command := m.startFilesystemCopy(entries, destination)
	if command != nil {
		t.Fatal("conflicts should open the dialog")
	}
	m = updated.(model)
	if m.modal != modalCopyConflict {
		t.Fatalf("modal=%v", m.modal)
	}
	m.copyConflictAction = copyConflictSkip
	m.copyConflictApplyAll = true
	updated, command = m.updateCopyConflict(tea.KeyMsg{Type: tea.KeyEnter})
	if command == nil {
		t.Fatal("apply-all decision should start copy operation")
	}
	message := command()
	updated, _ = updated.(model).Update(message)
	finished := updated.(model)
	if finished.modal != modalNone {
		t.Fatalf("modal=%v", finished.modal)
	}
	for _, name := range []string{"a.txt", "b.txt"} {
		content, _ := os.ReadFile(filepath.Join(destination, name))
		if string(content) != "old" {
			t.Fatalf("%s=%q", name, content)
		}
	}
}

func TestSuggestedCopyNameSkipsExistingCandidates(t *testing.T) {
	directory := t.TempDir()
	for _, name := range []string{"note.txt", "note (copy 1).txt"} {
		if err := os.WriteFile(filepath.Join(directory, name), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if got := suggestedCopyName(filepath.Join(directory, "note.txt")); got != "note (copy 2).txt" {
		t.Fatalf("suggestion=%q", got)
	}
}
