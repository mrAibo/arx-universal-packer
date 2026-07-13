package main

import (
	"fmt"
	"os"
	"path/filepath"
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
	} else if strings.Contains(strings.ToLower(status), "error") {
		statusLine = errorStyle.Width(width).Render(status)
	} else {
		statusLine = mutedStyle.Width(width).Render(status)
	}
	keyBar := m.renderKeyBar(width)

	base := strings.Join([]string{top, panels, statusLine, keyBar}, "\n")
	if m.modal == modalNone {
		return base
	}
	return base + "\n" + m.renderModal(width)
}

func (m model) renderPane(p pane, width, rows int, active bool) string {
	innerWidth := width - 2
	if innerWidth < 18 {
		innerWidth = 18
	}

	var b strings.Builder
	title := panelTitleStyle.Render(truncate(p.location(), innerWidth))
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
		line := renderEntryLine(item, innerWidth)
		if active && absoluteIndex == p.cursor {
			line = selectedStyle.Width(innerWidth).Render(line)
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
		{"1", "Help"},
		{"3", "Info"},
		{"5", "Pack/Xtr"},
		{"6", "Convert"},
		{"7", "Mkdir"},
		{"9", "Hidden"},
		{"10", "Quit"},
	}
	var b strings.Builder
	used := 0
	for _, item := range items {
		itemWidth := len([]rune(item[0])) + len([]rune(item[1])) + 2
		if used+itemWidth > width {
			break
		}
		b.WriteString(keyStyle.Render(item[0]))
		b.WriteString(keyLabelStyle.Render(" " + item[1] + " "))
		used += itemWidth
	}
	if used < width {
		b.WriteString(keyLabelStyle.Render(strings.Repeat(" ", width-used)))
	}
	return b.String()
}

func (m model) renderModal(width int) string {
	var body strings.Builder
	body.WriteString(panelTitleStyle.Render(m.modalTitle))
	body.WriteString("\n\n")

	switch m.modal {
	case modalHelp:
		body.WriteString("Tab          switch panel\n")
		body.WriteString("Enter/Right  open directory or archive\n")
		body.WriteString("Left/Backsp  parent directory / leave archive\n")
		body.WriteString("F5           pack selection or extract archive\n")
		body.WriteString("F6           convert selected archive\n")
		body.WriteString("F7           create directory\n")
		body.WriteString("F9 or .      show/hide dot files\n")
		body.WriteString("Ctrl-R       refresh panels\n")
		body.WriteString("F10 or q     quit\n\n")
		body.WriteString(mutedStyle.Render("Enter or Esc closes this help"))
	case modalMessage:
		body.WriteString(m.modalMessage)
		body.WriteString("\n\n")
		body.WriteString(mutedStyle.Render("Enter or Esc closes this message"))
	case modalFormat:
		for i, format := range archiveFormats {
			line := "  " + format
			if i == m.formatCursor {
				line = selectedStyle.Render(" ▸ " + format)
			}
			body.WriteString(line + "\n")
		}
		body.WriteString("\n" + mutedStyle.Render("↑/↓ choose · Enter confirm · Esc cancel"))
	}

	dialogWidth := 54
	if width < dialogWidth+4 {
		dialogWidth = width - 4
	}
	return dialogStyle.Width(dialogWidth).Render(body.String())
}

func renderEntryLine(item fileEntry, width int) string {
	nameWidth := width - 24
	if nameWidth < 8 {
		nameWidth = 8
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
	line := fmt.Sprintf("%-*s %8s %-16s", nameWidth, name, size, modified)
	line = truncate(line, width)

	if item.IsDir {
		return directoryStyle.Render(line)
	}
	if item.IsArchive {
		return archiveStyle.Render(line)
	}
	return line
}

func fitColumns(name, size, modified string, width int) string {
	nameWidth := width - 24
	if nameWidth < 8 {
		nameWidth = 8
	}
	return truncate(fmt.Sprintf("%-*s %8s %-16s", nameWidth, name, size, modified), width)
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

func availableArchiveName(directory, base, format string) string {
	if base == "" {
		base = "archive"
	}
	candidate := base
	for i := 1; ; i++ {
		path := filepath.Join(directory, candidate+"."+format)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			return candidate
		}
		candidate = fmt.Sprintf("%s-%d", base, i)
	}
}

func main() {
	if len(os.Args) > 1 {
		fmt.Println("This build is TUI-only. Run without args to use ARX Commander.")
		return
	}
	program := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := program.Run(); err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
}
