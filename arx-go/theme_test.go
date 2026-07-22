package main

import "testing"

func TestCycleThemeWrapsAndUpdatesStatus(t *testing.T) {
	m := initialModelAt(t.TempDir())
	for range len(colorThemes) {
		m.cycleTheme()
	}
	if m.themeIndex != 0 {
		t.Fatalf("theme index = %d, want 0", m.themeIndex)
	}
	if m.status != "Color theme: "+colorThemes[0].name {
		t.Fatalf("status = %q", m.status)
	}
}

func TestApplyThemeAcceptsWrappedIndex(t *testing.T) {
	applyTheme(-1)
	applyTheme(len(colorThemes))
}
