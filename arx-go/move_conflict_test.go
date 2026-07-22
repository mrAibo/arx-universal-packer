package main

import (
	"os"
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestMoveConflictSkipKeepsSource(t *testing.T) {
	sourceDir := t.TempDir()
	destination := t.TempDir()
	source := filepath.Join(sourceDir, "same.txt")
	if err := os.WriteFile(source, []byte("new"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(destination, "same.txt"), []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	plans, err := filesystemMoveDecisionPlans([]fileEntry{{Path: source}}, destination)
	if err != nil {
		t.Fatal(err)
	}
	if err := applyMoveDecision(plans, 0, copyConflictSkip, ""); err != nil {
		t.Fatal(err)
	}
	result := moveFilesystemDecisionPlans(plans, destination)
	if result.Err != nil {
		t.Fatal(result.Err)
	}
	if _, err := os.Stat(source); err != nil {
		t.Fatalf("source removed after skip: %v", err)
	}
	content, err := os.ReadFile(filepath.Join(destination, "same.txt"))
	if err != nil || string(content) != "old" {
		t.Fatalf("destination changed after skip: %q err=%v", content, err)
	}
}

func TestMoveConflictRenameMovesWithoutOverwrite(t *testing.T) {
	sourceDir := t.TempDir()
	destination := t.TempDir()
	source := filepath.Join(sourceDir, "same.txt")
	if err := os.WriteFile(source, []byte("new"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(destination, "same.txt"), []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	plans, err := filesystemMoveDecisionPlans([]fileEntry{{Path: source}}, destination)
	if err != nil {
		t.Fatal(err)
	}
	if err := applyMoveDecision(plans, 0, copyConflictRename, "renamed.txt"); err != nil {
		t.Fatal(err)
	}
	result := moveFilesystemDecisionPlans(plans, destination)
	if result.Err != nil {
		t.Fatal(result.Err)
	}
	if _, err := os.Stat(source); !os.IsNotExist(err) {
		t.Fatalf("source still exists after move: %v", err)
	}
	content, err := os.ReadFile(filepath.Join(destination, "renamed.txt"))
	if err != nil || string(content) != "new" {
		t.Fatalf("renamed content=%q err=%v", content, err)
	}
	content, err = os.ReadFile(filepath.Join(destination, "same.txt"))
	if err != nil || string(content) != "old" {
		t.Fatalf("original destination changed: %q err=%v", content, err)
	}
}

func TestMoveConflictApplyAllSkip(t *testing.T) {
	sourceDir := t.TempDir()
	destination := t.TempDir()
	entries := make([]fileEntry, 0, 2)
	for _, name := range []string{"a.txt", "b.txt"} {
		source := filepath.Join(sourceDir, name)
		if err := os.WriteFile(source, []byte("new"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(destination, name), []byte("old"), 0o644); err != nil {
			t.Fatal(err)
		}
		entries = append(entries, fileEntry{Path: source})
	}
	plans, err := filesystemMoveDecisionPlans(entries, destination)
	if err != nil {
		t.Fatal(err)
	}
	m := model{
		modal:                modalMoveConflict,
		movePlans:            plans,
		moveConflictIndex:    0,
		moveConflictAction:   copyConflictSkip,
		moveConflictApplyAll: true,
		moveConflictTarget:   destination,
	}
	updated, command := m.applyMoveConflictDecision()
	if command == nil {
		t.Fatal("apply-all did not start move")
	}
	message := command()
	updated, _ = updated.(model).Update(message)
	finished := updated.(model)
	if finished.modal != modalNone {
		t.Fatalf("unexpected modal: %v", finished.modal)
	}
	for _, entry := range entries {
		if _, err := os.Stat(entry.Path); err != nil {
			t.Fatalf("skipped source missing: %v", err)
		}
	}
}

func TestMoveConflictRejectsUnsafeRename(t *testing.T) {
	plans := []filesystemMoveDecisionPlan{{source: "/tmp/source", target: "/tmp/target", conflict: true}}
	if err := applyMoveDecision(plans, 0, copyConflictRename, "../escape"); err == nil {
		t.Fatal("expected unsafe rename error")
	}
}

func TestF6ConflictOpensMoveConflictDialog(t *testing.T) {
	sourceDir := t.TempDir()
	destination := t.TempDir()
	if err := os.WriteFile(filepath.Join(sourceDir, "same.txt"), []byte("new"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(destination, "same.txt"), []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	m := initialModelAt(sourceDir)
	m.panes[1] = newPane(destination)
	m.panes[0].selectName("same.txt")
	entries, err := m.panes[0].operationEntries()
	if err != nil {
		t.Fatal(err)
	}
	updated, command := m.startFilesystemMove(entries, destination, destination)
	if command != nil {
		t.Fatal("conflict should wait for a decision")
	}
	conflict := updated.(model)
	if conflict.modal != modalMoveConflict || conflict.moveConflictAction != copyConflictReplace {
		t.Fatalf("modal=%v action=%v", conflict.modal, conflict.moveConflictAction)
	}
	updated, command = conflict.updateMoveConflict(tea.KeyMsg{Type: tea.KeyEnter})
	if command == nil {
		t.Fatal("replace did not start move")
	}
	message := command()
	updated, _ = updated.(model).Update(message)
	finished := updated.(model)
	content, err := os.ReadFile(filepath.Join(destination, "same.txt"))
	if err != nil || string(content) != "new" {
		t.Fatalf("destination=%q err=%v modal=%v", content, err, finished.modal)
	}
}
