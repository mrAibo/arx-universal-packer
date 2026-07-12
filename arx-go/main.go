package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbletea"
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
	screen  screen
	cursor  int
	mode    string
	format  string
	source  string
	name    string
	target  string
	output  string
	err     string
	width   int
	height  int
}

var modes = []string{"Extract", "Compress", "List", "Convert"}
var formats = []string{"tar.gz", "tar.bz2", "tar.xz", "tar.zst", "zip", "7z", "tar"}

func initialModel() model {
	return model{screen: sMode, target: "."}
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			if m.screen == sDone {
				return m, tea.Quit
			}
			if m.screen == sMode {
				return m, tea.Quit
			}
			m.screen = sMode
			m.err = ""
		case "up", "k":
			m.move(-1)
		case "down", "j":
			m.move(1)
		case "enter":
			m = m.choose()
		case "backspace":
			if m.screen == sSource && len(m.source) > 0 {
				m.source = m.source[:len(m.source)-1]
			} else if m.screen == sName && len(m.name) > 0 {
				m.name = m.name[:len(m.name)-1]
			} else if m.screen == sTarget && len(m.target) > 0 {
				m.target = m.target[:len(m.target)-1]
			}
		default:
			// printable char into the active text field
			if len(msg.String()) == 1 {
				switch m.screen {
				case sSource:
					m.source += msg.String()
				case sName:
					m.name += msg.String()
				case sTarget:
					m.target += msg.String()
				}
			}
		}
	}
	return m, nil
}

func (m *model) move(d int) {
	list := m.options()
	n := len(list)
	m.cursor = (m.cursor + d + n) % n
}

func (m model) options() []string {
	switch m.screen {
	case sMode:
		return modes
	case sFormat:
		return formats
	}
	return nil
}

func (m model) choose() model {
	switch m.screen {
	case sMode:
		m.mode = strings.ToLower(modes[m.cursor])
		m.cursor = 0
		if m.mode == "extract" || m.mode == "list" {
			m.screen = sSource
		} else {
			m.screen = sFormat
		}
	case sFormat:
		m.format = formats[m.cursor]
		m.cursor = 0
		m.screen = sSource
	case sSource:
		if m.source == "" {
			m.err = "Type a file or folder path, then press Enter"
			return m
		}
		m.err = ""
		if m.mode == "compress" {
			m.screen = sName
		} else {
			m.run()
			m.screen = sDone
		}
	case sName:
		if m.name == "" {
			m.name = "archive"
		}
		m.screen = sTarget
	case sTarget:
		if m.target == "" {
			m.target = "."
		}
		m.run()
		m.screen = sDone
	case sDone:
		return m
	}
	return m
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
		// ponytail: convert needs a destination; reuse target+name+format
		dest := filepath.Join(m.target, m.name+"."+m.format)
		res = convert(m.source, dest)
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
		b.WriteString("\n" + helpStyle.Render("q quit · esc back to start"))
		return b.String()
	}
	return m.menuView()
}

func (m model) menuView() string {
	var b strings.Builder
	switch m.screen {
	case sMode:
		b.WriteString(titleStyle.Render("ARX - What do you want to do?"))
		b.WriteString("\n\n")
		b.WriteString(renderList(modes, m.cursor))
		b.WriteString("\n\n" + helpStyle.Render("↑/↓ select · enter confirm · q quit"))
	case sFormat:
		b.WriteString(titleStyle.Render("ARX - Choose format"))
		b.WriteString("\n\n")
		b.WriteString(renderList(formats, m.cursor))
		b.WriteString("\n\n" + helpStyle.Render("↑/↓ select · enter confirm · q back"))
	case sSource:
		b.WriteString(titleStyle.Render("ARX - " + strings.Title(m.mode) + " - which file or folder?"))
		b.WriteString("\n\n")
		b.WriteString(inputStyle.Render(m.source) + "\n")
		if m.err != "" {
			b.WriteString(errStyle.Render(m.err) + "\n")
		}
		b.WriteString("\n" + helpStyle.Render("type path · enter confirm · q back"))
	case sName:
		b.WriteString(titleStyle.Render("ARX - name the archive (no extension)"))
		b.WriteString("\n\n")
		b.WriteString(inputStyle.Render(m.name) + "\n")
		b.WriteString("\n" + helpStyle.Render("type name · enter confirm · q back"))
	case sTarget:
		b.WriteString(titleStyle.Render("ARX - where to put it? (folder)"))
		b.WriteString("\n\n")
		b.WriteString(inputStyle.Render(m.target) + "\n")
		b.WriteString("\n" + helpStyle.Render("type folder · enter GO · q back"))
	}
	return b.String()
}

func renderList(items []string, cur int) string {
	var b strings.Builder
	for i, it := range items {
		if i == cur {
			b.WriteString(selStyle.Render(" ▸ "+it) + "\n")
		} else {
			b.WriteString("   " + it + "\n")
		}
	}
	return b.String()
}

func main() {
	if len(os.Args) > 1 {
		// ponytail: non-interactive passthrough not needed; arx bash covers CLI.
		fmt.Println("This build is TUI-only. Run without args to use the interface.")
		return
	}
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
}
