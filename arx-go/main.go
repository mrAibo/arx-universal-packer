package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	titleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("63")).Padding(0, 1)
	helpStyle  = lipgloss.NewStyle().Faint(true)
	selStyle   = lipgloss.NewStyle().Background(lipgloss.Color("63")).Foreground(lipgloss.Color("0"))
	errStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
	okStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("46")).Bold(true)
	inputStyle = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 1)
)

type screen int

const (
	sMode screen = iota
	sFormat
	sSource
	sName
	sTarget
	sDone
)

type model struct {
	screen screen
	cursor int
	mode   string
	format string
	source string
	name   string
	target string
	output string
	err    string
	width  int
	height int
}

var modes = []string{"Extract", "Compress", "List", "Convert"}
var formats = []string{"tar.gz", "tar.bz2", "tar.xz", "tar.zst", "zip", "7z", "tar"}

func initialModel() model {
	return model{screen: sMode}
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
	case tea.KeyMsg:
		key := msg.String()
		if key == "ctrl+c" {
			return m, tea.Quit
		}
		if key == "esc" {
			if m.screen == sMode {
				return m, tea.Quit
			}
			return m.back(), nil
		}

		switch m.screen {
		case sMode, sFormat:
			return m.updateMenu(msg)
		case sSource, sName, sTarget:
			return m.updateInput(msg), nil
		case sDone:
			switch key {
			case "q":
				return m, tea.Quit
			case "enter":
				return initialModel(), nil
			}
		}
	}
	return m, nil
}

func (m model) updateMenu(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q":
		if m.screen == sMode {
			return m, tea.Quit
		}
		return m.back(), nil
	case "up", "k":
		m.move(-1)
	case "down", "j":
		m.move(1)
	case "enter":
		m = m.choose()
	}
	return m, nil
}

func (m model) updateInput(msg tea.KeyMsg) model {
	switch msg.String() {
	case "enter":
		return m.choose()
	case "backspace":
		m.deleteLastRune()
	case "up", "down", "left", "right", "home", "end":
		// Text editing is intentionally simple for now. Navigation keys must never
		// be interpreted as menu movement while an input field is active.
		return m
	default:
		if msg.Type == tea.KeyRunes {
			m.appendInput(string(msg.Runes))
		}
	}
	return m
}

func (m *model) appendInput(value string) {
	switch m.screen {
	case sSource:
		m.source += value
	case sName:
		m.name += value
	case sTarget:
		m.target += value
	}
}

func (m *model) deleteLastRune() {
	var value *string
	switch m.screen {
	case sSource:
		value = &m.source
	case sName:
		value = &m.name
	case sTarget:
		value = &m.target
	default:
		return
	}

	runes := []rune(*value)
	if len(runes) == 0 {
		return
	}
	*value = string(runes[:len(runes)-1])
}

func (m *model) move(delta int) {
	n := len(m.options())
	if n == 0 {
		return
	}
	m.cursor = (m.cursor + delta) % n
	if m.cursor < 0 {
		m.cursor += n
	}
}

func (m model) options() []string {
	switch m.screen {
	case sMode:
		return modes
	case sFormat:
		return formats
	default:
		return nil
	}
}

func (m model) choose() model {
	switch m.screen {
	case sMode:
		if m.cursor < 0 || m.cursor >= len(modes) {
			m.cursor = 0
		}
		m.mode = strings.ToLower(modes[m.cursor])
		m.cursor = 0
		if m.mode == "extract" || m.mode == "list" {
			m.screen = sSource
		} else {
			m.screen = sFormat
		}
	case sFormat:
		if m.cursor < 0 || m.cursor >= len(formats) {
			m.cursor = 0
		}
		m.format = formats[m.cursor]
		m.cursor = 0
		m.screen = sSource
	case sSource:
		if strings.TrimSpace(m.source) == "" {
			m.err = "Type a file or folder path, then press Enter"
			return m
		}
		m.err = ""
		switch m.mode {
		case "list":
			m.run()
			m.screen = sDone
		case "extract":
			m.screen = sTarget
		case "compress", "convert":
			m.screen = sName
		default:
			m.err = "Unknown operation"
		}
	case sName:
		if strings.TrimSpace(m.name) == "" {
			m.name = defaultOutputName(m.source)
		}
		m.screen = sTarget
	case sTarget:
		if strings.TrimSpace(m.target) == "" {
			m.target = "."
		}
		m.run()
		m.screen = sDone
	case sDone:
		return m
	}
	return m
}

func (m model) back() model {
	m.err = ""
	switch m.screen {
	case sFormat:
		m.screen = sMode
		m.cursor = optionIndex(modes, m.mode)
	case sSource:
		if m.mode == "compress" || m.mode == "convert" {
			m.screen = sFormat
			m.cursor = optionIndex(formats, m.format)
		} else {
			m.screen = sMode
			m.cursor = optionIndex(modes, m.mode)
		}
	case sName:
		m.screen = sSource
	case sTarget:
		if m.mode == "extract" {
			m.screen = sSource
		} else {
			m.screen = sName
		}
	case sDone:
		return initialModel()
	}
	return m
}

func optionIndex(options []string, selected string) int {
	for i, option := range options {
		if strings.EqualFold(option, selected) {
			return i
		}
	}
	return 0
}

func defaultOutputName(source string) string {
	base := filepath.Base(filepath.Clean(source))
	lower := strings.ToLower(base)
	for _, suffix := range []string{".tar.gz", ".tar.bz2", ".tar.xz", ".tar.zst", ".tgz", ".tbz2", ".txz", ".zip", ".7z", ".tar"} {
		if strings.HasSuffix(lower, suffix) {
			base = base[:len(base)-len(suffix)]
			break
		}
	}
	if base == "" || base == "." || base == string(filepath.Separator) {
		return "archive"
	}
	return base
}

func (m *model) run() {
	var res Result
	switch m.mode {
	case "extract":
		res = extract(m.source, m.target)
	case "list":
		res = list(m.source)
	case "compress":
		res = compress(m.format, m.name, m.source, m.target, 3)
	case "convert":
		dest := filepath.Join(m.target, m.name+"."+m.format)
		res = convert(m.source, dest)
	default:
		res.Err = fmt.Errorf("unknown operation: %s", m.mode)
	}
	if res.Err != nil {
		m.err = res.Err.Error()
		m.output = ""
	} else {
		m.output = res.Output
		m.err = ""
	}
}

func (m model) View() string {
	switch m.screen {
	case sDone:
		var b strings.Builder
		b.WriteString(titleStyle.Render("ARX - Done"))
		b.WriteString("\n\n")
		if m.err != "" {
			b.WriteString(errStyle.Render("Error: "+m.err) + "\n")
		} else {
			b.WriteString(okStyle.Render(m.output) + "\n")
		}
		b.WriteString("\n" + helpStyle.Render("q quit · enter/esc back to start"))
		return b.String()
	default:
		return m.menuView()
	}
}

func (m model) menuView() string {
	var b strings.Builder
	switch m.screen {
	case sMode:
		b.WriteString(titleStyle.Render("ARX - What do you want to do?"))
		b.WriteString("\n\n")
		b.WriteString(renderList(modes, m.cursor))
		b.WriteString("\n\n" + helpStyle.Render("↑/↓ select · enter confirm · q/esc quit"))
	case sFormat:
		b.WriteString(titleStyle.Render("ARX - Choose format"))
		b.WriteString("\n\n")
		b.WriteString(renderList(formats, m.cursor))
		b.WriteString("\n\n" + helpStyle.Render("↑/↓ select · enter confirm · q/esc back"))
	case sSource:
		b.WriteString(titleStyle.Render("ARX - " + modeTitle(m.mode) + " - which file or folder?"))
		b.WriteString("\n\n")
		b.WriteString(inputStyle.Render(m.source) + "\n")
		if m.err != "" {
			b.WriteString(errStyle.Render(m.err) + "\n")
		}
		b.WriteString("\n" + helpStyle.Render("type path · enter confirm · esc back"))
	case sName:
		b.WriteString(titleStyle.Render("ARX - name the archive (no extension)"))
		b.WriteString("\n\n")
		b.WriteString(inputStyle.Render(m.name) + "\n")
		b.WriteString("\n" + helpStyle.Render("type name · enter confirm · esc back"))
	case sTarget:
		b.WriteString(titleStyle.Render("ARX - where to put it? (folder)"))
		b.WriteString("\n\n")
		b.WriteString(inputStyle.Render(m.target) + "\n")
		b.WriteString("\n" + helpStyle.Render("empty = current folder · enter GO · esc back"))
	}
	return b.String()
}

func modeTitle(mode string) string {
	if mode == "" {
		return ""
	}
	return strings.ToUpper(mode[:1]) + mode[1:]
}

func renderList(items []string, cur int) string {
	var b strings.Builder
	for i, item := range items {
		if i == cur {
			b.WriteString(selStyle.Render(" ▸ "+item) + "\n")
		} else {
			b.WriteString("   " + item + "\n")
		}
	}
	return b.String()
}

func main() {
	if len(os.Args) > 1 {
		fmt.Println("This build is TUI-only. Run without args to use the interface.")
		return
	}
	program := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := program.Run(); err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
}
