package main

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func (m model) View() string {
	width := m.width
	if width < 60 {
		width = 60
	}
	panelWidth := (width - 1) / 2
	rows := m.panelRows()

	top := menuStyle.Width(width).Render("  Left   File   Command   Options   Right")
	left := m.renderPane(m.panes[0], panelWidth, rows, m.active == 0)
	right := m.renderPane(m.panes[1], width-panelWidth, rows, m.active == 1)
	panels := lipgloss.JoinHorizontal(lipgloss.Top, left, right)

	status := truncate(m.status, width)
	var statusLine string
	if m.busy {
		statusLine = busyStyle.Width(width).Render(truncate("WORKING  "+m.status, width))
	} else if strings.Contains(strings.ToLower(status), "error") || strings.Contains(strings.ToLower(status), "failed") {
		statusLine = errorStyle.Width(width).Render(status)
	} else {
		statusLine = mutedStyle.Width(width).Render(status)
	}
	keyBar := m.renderKeyBar(width)

	base := strings.Join([]string{top, panels, statusLine, keyBar}, "\n")
	if m.modal == modalNone {
		return base
	}
	if m.modal == modalViewer {
		return base + "\n" + m.renderViewerModal(width)
	}
	return base + "\n" + m.renderModal(width)
}

func (m model) renderPane(p pane, width, rows int, active bool) string {
	innerWidth := width - 2
	if innerWidth < 18 {
		innerWidth = 18
	}

	var b strings.Builder
	titleText := p.location()
	if len(p.markedEntries()) > 0 {
		titleText += fmt.Sprintf("  [%d marked]", len(p.markedEntries()))
	}
	title := panelTitleStyle.Render(truncate(titleText, innerWidth))
	b.WriteString(title)
	b.WriteString("\n")
	b.WriteString(mutedStyle.Render(fitColumns("Name", "Size", "Modified", innerWidth)))
	b.WriteString("\n")

	visible := p.visibleEntries(rows)
	for row := 0; row < rows; row++ {
		if row >= len(visible) {
			b.WriteString(strings.Repeat(" ", innerWidth))
			if row < rows-1 {
				b.WriteString("\n")
			}
			continue
		}
		item := visible[row]
		absoluteIndex := p.offset + row
		text := renderEntryText(item, innerWidth, p.isMarked(item))
		line := text
		switch {
		case active && absoluteIndex == p.cursor:
			line = selectedStyle.Width(innerWidth).Render(text)
		case p.isMarked(item):
			line = markedStyle.Width(innerWidth).Render(text)
		case item.IsDir:
			line = directoryStyle.Render(text)
		case item.IsArchive:
			line = archiveStyle.Render(text)
		}
		b.WriteString(line)
		if row < rows-1 {
			b.WriteString("\n")
		}
	}

	style := panelBorderStyle.Width(width - 2)
	if active {
		style = activePanelBorderStyle.Width(width - 2)
	}
	return style.Render(b.String())
}

func (m model) renderKeyBar(width int) string {
	items := [][2]string{
		{"F1", "Help"},
		{"F2", "Archive"},
		{"F3", "View"},
		{"F4", "Test"},
		{"F5", m.f5Label()},
		{"F6", "Conv"},
		{"F7", "Mkdir"},
		{"F8", m.f8Label()},
		{"F9", "Menu"},
		{"F10", "Quit"},
	}
	var b strings.Builder
	used := 0
	for _, item := range items {
		key := keyStyle.Render(item[0])
		label := keyLabelStyle.Render(" " + item[1] + " ")
		itemWidth := lipgloss.Width(key) + lipgloss.Width(label)
		if used+itemWidth > width {
			break
		}
		b.WriteString(key)
		b.WriteString(label)
		used += itemWidth
	}
	if used < width {
		b.WriteString(keyLabelStyle.Render(strings.Repeat(" ", width-used)))
	}
	return b.String()
}

func (m model) f5Label() string {
	active := m.panes[m.active]
	passive := m.panes[1-m.active]
	if active.mode == paneArchive {
		return "Extract"
	}
	if passive.mode == paneArchive {
		return "Add"
	}
	if len(active.markedEntries()) == 0 {
		if entry, ok := active.selected(); ok && entry.IsArchive && !entry.IsDir {
			return "Extract"
		}
	}
	return "Copy"
}

func (m model) f8Label() string {
	if m.panes[m.active].mode == paneArchive {
		return "Delete"
	}
	return "Clear"
}

func (m model) renderModal(width int) string {
	var body strings.Builder
	body.WriteString(panelTitleStyle.Render(m.modalTitle))
	body.WriteString("\n\n")

	switch m.modal {
	case modalHelp:
		body.WriteString("Tab             switch panel\n")
		body.WriteString("Enter/Right     open directory or archive\n")
		body.WriteString("Left/Backspace  parent directory / leave archive\n")
		body.WriteString("Space/Insert    mark or unmark current item\n")
		body.WriteString("F2              create archive from selected filesystem items\n")
		body.WriteString("Ctrl-A           mark all visible items\n")
		body.WriteString("*                invert marks\n")
		body.WriteString("Ctrl-U           clear marks\n")
		body.WriteString("F3              view file or archive member\n")
		body.WriteString("F4              test selected archive\n")
		body.WriteString("F5              copy/extract/add according to panel direction\n")
		body.WriteString("F6              convert selected archive\n")
		body.WriteString("F7              create a named directory\n")
		body.WriteString("F8              delete archive entries; clear filesystem marks\n")
		body.WriteString("F9              open command menu\n")
		body.WriteString("Ctrl-H or .      show/hide dot files\n")
		body.WriteString("Ctrl-L / Alt-C   change directory\n")
		body.WriteString("Ctrl-S / Alt-S   quick search; Ctrl-S repeats\n")
		body.WriteString("Alt-Y / Alt-U    history back / forward\n")
		body.WriteString("Alt-H            directory history\n")
		body.WriteString("Ctrl-\\          favorites; Ctrl-B adds current location\n")
		body.WriteString("Ctrl-R          refresh panels\n")
		body.WriteString("F10 or q        quit\n\n")
		body.WriteString("F2 action:\n")
		body.WriteString("  filesystem → filesystem   create a new archive\n\n")
		body.WriteString("F5 direction:\n")
		body.WriteString("  filesystem → filesystem   copy selected entries\n")
		body.WriteString("  archive → filesystem      extract selected entries\n")
		body.WriteString("  filesystem → archive      add selected entries\n\n")
		body.WriteString("Mouse: click selects, double-click opens, right/middle click marks, wheel scrolls.\n\n")
		body.WriteString(mutedStyle.Render("Enter or Esc closes this help"))
	case modalMessage:
		body.WriteString(m.modalMessage)
		body.WriteString("\n\n")
		body.WriteString(mutedStyle.Render("Enter or Esc closes this message"))
	case modalArchive:
		body.WriteString(m.renderArchiveDialog())
	case modalConfirm:
		body.WriteString(m.modalMessage)
		body.WriteString("\n\n")
		body.WriteString(errorStyle.Render("Enter/Y confirms"))
		body.WriteString("   ")
		body.WriteString(mutedStyle.Render("Esc/N cancels"))
	case modalNavigationMenu, modalNavigationInput, modalNavigationList:
		body.WriteString(m.renderNavigationModal())
	}

	dialogWidth := 62
	if width < dialogWidth+4 {
		dialogWidth = width - 4
	}
	if dialogWidth < 30 {
		dialogWidth = 30
	}
	return dialogStyle.Width(dialogWidth).Render(body.String())
}

func (m model) renderViewerModal(width int) string {
	dialogWidth := width - 4
	if dialogWidth < 50 {
		dialogWidth = 50
	}
	contentWidth := dialogWidth - 8
	if contentWidth < 20 {
		contentWidth = 20
	}
	rows := m.viewerRows()

	var body strings.Builder
	body.WriteString(panelTitleStyle.Render(truncate(m.modalTitle, contentWidth)))
	body.WriteString("\n")
	body.WriteString(mutedStyle.Render(fmt.Sprintf("Line %d/%d · Column %d · arrows/PageUp/PageDown scroll · F3/Esc close", minInt(m.viewerOffset+1, len(m.viewerLines)), len(m.viewerLines), m.viewerColumn+1)))
	body.WriteString("\n\n")

	for row := 0; row < rows; row++ {
		index := m.viewerOffset + row
		if index < len(m.viewerLines) {
			line := sliceRunes(m.viewerLines[index], m.viewerColumn, contentWidth-8)
			body.WriteString(fmt.Sprintf("%6d  %s", index+1, line))
		}
		if row < rows-1 {
			body.WriteString("\n")
		}
	}
	return dialogStyle.Width(dialogWidth).Render(body.String())
}

func (m model) renderArchiveDialog() string {
	var body strings.Builder
	destination := m.panes[1-m.active].path

	body.WriteString(fmt.Sprintf("Source items: %d\n", len(m.pendingSources)))
	body.WriteString("Destination:  " + destination + "\n\n")

	nameValue := m.archiveName
	if nameValue == "" {
		nameValue = " "
	}
	nameBox := fieldStyle.Width(42).Render(nameValue)
	if m.dialogField == dialogName {
		nameBox = activeFieldStyle.Width(42).Render(nameValue)
	}
	body.WriteString("Name\n" + nameBox + "\n\n")

	formatValue := "◀  " + archiveFormats[m.formatCursor] + "  ▶"
	formatBox := fieldStyle.Width(24).Render(formatValue)
	if m.dialogField == dialogFormat {
		formatBox = activeFieldStyle.Width(24).Render(formatValue)
	}
	body.WriteString("Format\n" + formatBox + "\n\n")

	levelValue := fmt.Sprintf("◀  %d  ▶", m.compression)
	if archiveFormats[m.formatCursor] == "tar" {
		levelValue = "not used for tar"
	}
	levelBox := fieldStyle.Width(24).Render(levelValue)
	if m.dialogField == dialogLevel {
		levelBox = activeFieldStyle.Width(24).Render(levelValue)
	}
	body.WriteString("Compression level\n" + levelBox + "\n\n")

	button := fieldStyle.Render(" Create ")
	if m.dialogField == dialogCreate {
		button = activeFieldStyle.Render(" Create ")
	}
	body.WriteString(button + "   " + mutedStyle.Render("Esc Cancel"))

	if m.dialogError != "" {
		body.WriteString("\n\n" + errorStyle.Render(m.dialogError))
	}
	body.WriteString("\n\n" + mutedStyle.Render("Tab changes field · arrows change values · F5 creates · Ctrl-U clears name"))
	return body.String()
}

func renderEntryText(item fileEntry, width int, marked bool) string {
	nameWidth := width - 26
	if nameWidth < 6 {
		nameWidth = 6
	}
	marker := "  "
	if marked {
		marker = "* "
	}
	name := item.Name
	if item.IsDir && name != ".." {
		name += "/"
	}
	name = truncate(name, nameWidth)

	size := formatSize(item.Size)
	if item.IsDir {
		size = "<DIR>"
	}
	modified := ""
	if !item.ModTime.IsZero() {
		modified = item.ModTime.Format("2006-01-02 15:04")
	}
	line := fmt.Sprintf("%s%-*s %8s %-16s", marker, nameWidth, name, size, modified)
	return truncate(line, width)
}

func fitColumns(name, size, modified string, width int) string {
	nameWidth := width - 26
	if nameWidth < 6 {
		nameWidth = 6
	}
	return truncate(fmt.Sprintf("  %-*s %8s %-16s", nameWidth, name, size, modified), width)
}

func formatSize(size int64) string {
	if size < 0 {
		return ""
	}
	units := []string{"B", "K", "M", "G", "T"}
	value := float64(size)
	unit := 0
	for value >= 1024 && unit < len(units)-1 {
		value /= 1024
		unit++
	}
	if unit == 0 {
		return fmt.Sprintf("%d%s", size, units[unit])
	}
	return fmt.Sprintf("%.1f%s", value, units[unit])
}

func truncate(value string, width int) string {
	if width <= 0 {
		return ""
	}
	runes := []rune(value)
	if len(runes) <= width {
		return value
	}
	if width == 1 {
		return "…"
	}
	return string(runes[:width-1]) + "…"
}

func sliceRunes(value string, start, width int) string {
	if width <= 0 {
		return ""
	}
	runes := []rune(value)
	if start >= len(runes) {
		return ""
	}
	if start < 0 {
		start = 0
	}
	end := start + width
	if end > len(runes) {
		end = len(runes)
	}
	return string(runes[start:end])
}

func minInt(left, right int) int {
	if left < right {
		return left
	}
	return right
}

func wrapIndex(value, count int) int {
	if count <= 0 {
		return 0
	}
	value %= count
	if value < 0 {
		value += count
	}
	return value
}

func indexOf(values []string, wanted string) int {
	for i, value := range values {
		if value == wanted {
			return i
		}
	}
	return 0
}

func main() {
	if len(os.Args) > 1 {
		fmt.Println("This build is TUI-only. Run without args to use ARX Commander.")
		return
	}
	program := tea.NewProgram(
		initialModel(),
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)
	if _, err := program.Run(); err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
}
