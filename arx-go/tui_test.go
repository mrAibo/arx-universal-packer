package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/charmbracelet/bubbletea"
)

// key builds a tea.KeyMsg from a string (same as bubbletea's).
func key(s string) tea.Msg { return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)} }

func TestTUIDriveCompress(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src")
	os.Mkdir(src, 0o755)
	os.WriteFile(filepath.Join(src, "f.txt"), []byte("x"), 0o644)

	m := initialModel()
	step := func(msg tea.Msg) { mm, _ := m.Update(msg); m = mm.(model) }
	// mode menu -> down to Compress (index 1), enter
	step(key("down"))
	step(key("enter"))
	if m.mode != "compress" {
		t.Fatalf("mode=%q want compress", m.mode)
	}
	if m.screen != sFormat {
		t.Fatalf("screen=%v want sFormat", m.screen)
	}
	// format -> enter (tar.gz)
	step(key("enter"))
	if m.format != "tar.gz" {
		t.Fatalf("format=%q", m.format)
	}
	if m.screen != sSource {
		t.Fatalf("screen=%v want sSource", m.screen)
	}
	// source: type path
	for _, r := range src {
		step(key(string(r)))
	}
	step(key("enter"))
	if m.screen != sName {
		t.Fatalf("screen=%v want sName", m.screen)
	}
	// name
	for _, r := range "demo" {
		step(key(string(r)))
	}
	step(key("enter"))
	if m.screen != sTarget {
		t.Fatalf("screen=%v want sTarget", m.screen)
	}
	// target: default ".", run
	step(key("enter"))
	if m.screen != sDone {
		t.Fatalf("screen=%v want sDone", m.screen)
	}
	if m.err != "" {
		t.Fatalf("err=%q", m.err)
	}
	arc := filepath.Join(".", "demo.tar.gz")
	if _, err := os.Stat(arc); err != nil {
		t.Fatalf("archive not created: %v", err)
	}
	os.Remove(arc)
}

func TestTUIBackspace(t *testing.T) {
	m := initialModel()
	m.screen = sSource
	m.source = "abc"
	mm, _ := m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	m = mm.(model)
	if m.source != "ab" {
		t.Fatalf("backspace failed: %q", m.source)
	}
}
