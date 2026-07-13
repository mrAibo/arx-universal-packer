package main

import (
	"os"
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func key(s string) tea.Msg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}

func TestTUIDriveCompress(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src")
	if err := os.Mkdir(src, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "f.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	m := initialModel()
	step := func(msg tea.Msg) {
		updated, _ := m.Update(msg)
		m = updated.(model)
	}

	step(key("down"))
	step(key("enter"))
	if m.mode != "compress" {
		t.Fatalf("mode=%q want compress", m.mode)
	}
	if m.screen != sFormat {
		t.Fatalf("screen=%v want sFormat", m.screen)
	}

	step(key("enter"))
	if m.format != "tar.gz" {
		t.Fatalf("format=%q want tar.gz", m.format)
	}
	if m.screen != sSource {
		t.Fatalf("screen=%v want sSource", m.screen)
	}

	for _, r := range src {
		step(key(string(r)))
	}
	step(key("enter"))
	if m.screen != sName {
		t.Fatalf("screen=%v want sName", m.screen)
	}

	for _, r := range "demo" {
		step(key(string(r)))
	}
	step(key("enter"))
	if m.screen != sTarget {
		t.Fatalf("screen=%v want sTarget", m.screen)
	}

	step(key("enter"))
	if m.screen != sDone {
		t.Fatalf("screen=%v want sDone", m.screen)
	}
	if m.err != "" {
		t.Fatalf("err=%q", m.err)
	}

	archive := filepath.Join(".", "demo.tar.gz")
	if _, err := os.Stat(archive); err != nil {
		t.Fatalf("archive not created: %v", err)
	}
	if err := os.Remove(archive); err != nil {
		t.Fatal(err)
	}
}

func TestInputScreensIgnoreMenuNavigation(t *testing.T) {
	for _, screen := range []screen{sSource, sName, sTarget, sDone} {
		t.Run(screenName(screen), func(t *testing.T) {
			m := initialModel()
			m.screen = screen
			m.cursor = 7

			for _, msg := range []tea.Msg{
				tea.KeyMsg{Type: tea.KeyUp},
				tea.KeyMsg{Type: tea.KeyDown},
			} {
				updated, _ := m.Update(msg)
				m = updated.(model)
			}

			if m.cursor != 7 {
				t.Fatalf("cursor changed on non-menu screen: got %d want 7", m.cursor)
			}
		})
	}
}

func TestInputAcceptsMenuShortcutRunesAndUnicode(t *testing.T) {
	m := initialModel()
	m.screen = sSource

	for _, value := range []string{"q", "j", "k", "ä", "Я"} {
		updated, _ := m.Update(key(value))
		m = updated.(model)
	}

	if m.source != "qjkäЯ" {
		t.Fatalf("source=%q want %q", m.source, "qjkäЯ")
	}
}

func TestBackspaceRemovesWholeUnicodeRune(t *testing.T) {
	m := initialModel()
	m.screen = sSource
	m.source = "abcЯ"

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	m = updated.(model)

	if m.source != "abc" {
		t.Fatalf("backspace failed: got %q want abc", m.source)
	}
}

func TestExtractPromptsForTargetBeforeRunning(t *testing.T) {
	m := initialModel()
	m = m.choose()
	if m.mode != "extract" || m.screen != sSource {
		t.Fatalf("unexpected extract start: mode=%q screen=%v", m.mode, m.screen)
	}

	m.source = "archive.tar.gz"
	m = m.choose()
	if m.screen != sTarget {
		t.Fatalf("screen=%v want sTarget", m.screen)
	}
}

func TestConvertPromptsForNameAndTarget(t *testing.T) {
	m := initialModel()
	m.cursor = 3
	m = m.choose()
	if m.mode != "convert" || m.screen != sFormat {
		t.Fatalf("unexpected convert start: mode=%q screen=%v", m.mode, m.screen)
	}

	m = m.choose()
	m.source = "backup.tar.gz"
	m = m.choose()
	if m.screen != sName {
		t.Fatalf("screen=%v want sName", m.screen)
	}

	m = m.choose()
	if m.name != "backup" {
		t.Fatalf("default name=%q want backup", m.name)
	}
	if m.screen != sTarget {
		t.Fatalf("screen=%v want sTarget", m.screen)
	}
}

func TestMenuMovementWraps(t *testing.T) {
	m := initialModel()
	m.move(-1)
	if m.cursor != len(modes)-1 {
		t.Fatalf("cursor=%d want %d", m.cursor, len(modes)-1)
	}
	m.move(1)
	if m.cursor != 0 {
		t.Fatalf("cursor=%d want 0", m.cursor)
	}
}

func screenName(value screen) string {
	switch value {
	case sMode:
		return "mode"
	case sFormat:
		return "format"
	case sSource:
		return "source"
	case sName:
		return "name"
	case sTarget:
		return "target"
	case sDone:
		return "done"
	default:
		return "unknown"
	}
}
