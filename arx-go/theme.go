package main

import "github.com/charmbracelet/lipgloss"

type colorTheme struct {
	name         string
	menu         string
	menuText     string
	border       string
	active       string
	title        string
	selectedText string
	marked       string
	markedText   string
	directory    string
	archive      string
	muted        string
	error        string
	busy         string
	keyLabel     string
	keyLabelText string
}

var colorThemes = []colorTheme{
	{name: "Midnight", menu: "19", menuText: "231", border: "240", active: "39", title: "45", selectedText: "231", marked: "58", markedText: "229", directory: "45", archive: "220", muted: "244", error: "196", busy: "220", keyLabel: "252", keyLabelText: "0"},
	{name: "Nord", menu: "24", menuText: "255", border: "60", active: "110", title: "117", selectedText: "16", marked: "67", markedText: "255", directory: "110", archive: "179", muted: "109", error: "167", busy: "179", keyLabel: "253", keyLabelText: "16"},
	{name: "Forest", menu: "22", menuText: "255", border: "65", active: "42", title: "48", selectedText: "16", marked: "100", markedText: "230", directory: "48", archive: "214", muted: "108", error: "203", busy: "214", keyLabel: "254", keyLabelText: "16"},
	{name: "Monochrome", menu: "235", menuText: "255", border: "243", active: "255", title: "255", selectedText: "16", marked: "250", markedText: "16", directory: "255", archive: "252", muted: "245", error: "255", busy: "255", keyLabel: "252", keyLabelText: "16"},
}

func applyTheme(index int) {
	if len(colorThemes) == 0 {
		return
	}
	index = wrapIndex(index, len(colorThemes))
	theme := colorThemes[index]

	menuStyle = lipgloss.NewStyle().Background(lipgloss.Color(theme.menu)).Foreground(lipgloss.Color(theme.menuText)).Bold(true)
	panelBorderStyle = lipgloss.NewStyle().Border(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color(theme.border))
	activePanelBorderStyle = lipgloss.NewStyle().Border(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color(theme.active))
	panelTitleStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(theme.title)).Bold(true)
	selectedStyle = lipgloss.NewStyle().Background(lipgloss.Color(theme.active)).Foreground(lipgloss.Color(theme.selectedText))
	markedStyle = lipgloss.NewStyle().Background(lipgloss.Color(theme.marked)).Foreground(lipgloss.Color(theme.markedText)).Bold(true)
	directoryStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(theme.directory)).Bold(true)
	archiveStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(theme.archive))
	mutedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(theme.muted))
	errorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(theme.error)).Bold(true)
	busyStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(theme.busy)).Bold(true)
	keyStyle = lipgloss.NewStyle().Background(lipgloss.Color(theme.menu)).Foreground(lipgloss.Color(theme.menuText)).Bold(true)
	keyLabelStyle = lipgloss.NewStyle().Background(lipgloss.Color(theme.keyLabel)).Foreground(lipgloss.Color(theme.keyLabelText))
	dialogStyle = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color(theme.active)).Padding(1, 2)
	fieldStyle = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color(theme.border)).Padding(0, 1)
	activeFieldStyle = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color(theme.active)).Padding(0, 1)
}

func (m *model) cycleTheme() {
	m.themeIndex = wrapIndex(m.themeIndex+1, len(colorThemes))
	applyTheme(m.themeIndex)
	m.status = "Color theme: " + colorThemes[m.themeIndex].name
}
