package main

import (
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const defaultArchiveFormat = "tar.zst"

var archiveFormats = []string{"tar.zst", "tar.gz", "tar.xz", "tar.bz2", "zip", "7z", "tar"}

var (
	menuStyle = lipgloss.NewStyle().
		Background(lipgloss.Color("19")).
		Foreground(lipgloss.Color("231")).
		Bold(true)

	panelBorderStyle = lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240"))

	activePanelBorderStyle = lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("39"))

	panelTitleStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("45")).
		Bold(true)

	selectedStyle = lipgloss.NewStyle().
		Background(lipgloss.Color("39")).
		Foreground(lipgloss.Color("231"))

	directoryStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("45")).
		Bold(true)

	archiveStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("220"))

	mutedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	errorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
	busyStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("220")).Bold(true)

	keyStyle = lipgloss.NewStyle().
		Background(lipgloss.Color("19")).
		Foreground(lipgloss.Color("231")).
		Bold(true)

	keyLabelStyle = lipgloss.NewStyle().
		Background(lipgloss.Color("252")).
		Foreground(lipgloss.Color("0"))

	dialogStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("39")).
		Padding(1, 2)
)

type modalKind int

const (
	modalNone modalKind = iota
	modalHelp
	modalFormat
	modalMessage
)

type pendingAction int

const (
	actionNone pendingAction = iota
	actionPack
	actionConvert
)

type operationMsg struct {
	result Result
}

type model struct {
	panes        [2]pane
	active       int
	width        int
	height       int
	status       string
	busy         bool
	modal        modalKind
	modalTitle   string
	modalMessage string
	formatCursor int
	pending      pendingAction
	pendingPath  string
}

func initialModel() model {
	cwd, err := os.Getwd()
	if err != nil {
		cwd = "."
	}
	return initialModelAt(cwd)
}

func initialModelAt(path string) model {
	left := newPane(path)
	right := newPane(path)
	m := model{
		panes:  [2]pane{left, right},
		active: 0,
		width:  100,
		height: 30,
		status: "Ready",
	}
	if left.err != "" {
		m.status = left.err
	}
	return m
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ensureVisible()
		return m, nil
	case operationMsg:
		m.busy = false
		if msg.result.Err != nil {
			m.modal = modalMessage
			m.modalTitle = "Operation failed"
			m.modalMessage = msg.result.Err.Error()
			m.status = msg.result.Err.Error()
		} else {
			m.status = msg.result.Output
			m.reloadPanes()
		}
		return m, nil
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" || msg.String() == "f10" {
			return m, tea.Quit
		}
		if m.modal != modalNone {
			return m.updateModal(msg)
		}
		if m.busy {
			return m, nil
		}
		return m.updateBrowser(msg)
	default:
		return m, nil
	}
}

func (m model) updateBrowser(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	active := &m.panes[m.active]
	rows := m.panelRows()

	switch msg.String() {
	case "q":
		return m, tea.Quit
	case "tab":
		m.active = 1 - m.active
	case "up", "k":
		active.move(-1, rows)
	case "down", "j":
		active.move(1, rows)
	case "pgup":
		active.move(-rows, rows)
	case "pgdown":
		active.move(rows, rows)
	case "home", "g":
		active.cursor = 0
		active.ensureVisible(rows)
	case "end", "G":
		if len(active.entries) > 0 {
			active.cursor = len(active.entries) - 1
			active.ensureVisible(rows)
		}
	case "enter", "right":
		if err := active.openSelected(); err != nil {
			m.showError(err)
		} else {
			m.status = active.location()
		}
	case "left", "backspace":
		if err := active.goUp(); err != nil {
			m.showError(err)
		} else {
			m.status = active.location()
		}
	case "f1":
		m.modal = modalHelp
		m.modalTitle = "ARX Commander help"
	case "f3":
		m.status = active.selectedDescription()
	case "f5":
		return m.startF5()
	case "f6":
		return m.startConvert()
	case "f7":
		if err := active.createDirectory(); err != nil {
			m.showError(err)
		} else {
			m.status = "Directory created"
		}
	case "f9", ".":
		active.showHidden = !active.showHidden
		if err := active.reload(); err != nil {
			m.showError(err)
		} else if active.showHidden {
			m.status = "Hidden files: shown"
		} else {
			m.status = "Hidden files: hidden"
		}
	case "ctrl+r", "r":
		m.reloadPanes()
		m.status = "Panels refreshed"
	}

	m.ensureVisible()
	return m, nil
}

func (m model) updateModal(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.modal {
	case modalHelp, modalMessage:
		switch msg.String() {
		case "esc", "enter", "q", "f1":
			m.closeModal()
		}
		return m, nil
	case modalFormat:
		switch msg.String() {
		case "esc", "q":
			m.closeModal()
			return m, nil
		case "up", "k":
			m.formatCursor = wrapIndex(m.formatCursor-1, len(archiveFormats))
		case "down", "j":
			m.formatCursor = wrapIndex(m.formatCursor+1, len(archiveFormats))
		case "enter":
			format := archiveFormats[m.formatCursor]
			action := m.pending
			path := m.pendingPath
			m.closeModal()
			return m.runPending(action, path, format)
		}
	}
	return m, nil
}

func (m *model) closeModal() {
	m.modal = modalNone
	m.modalTitle = ""
	m.modalMessage = ""
	m.pending = actionNone
	m.pendingPath = ""
}

func (m *model) showError(err error) {
	m.modal = modalMessage
	m.modalTitle = "Error"
	m.modalMessage = err.Error()
	m.status = err.Error()
}

func (m model) startF5() (tea.Model, tea.Cmd) {
	active := m.panes[m.active]
	passive := m.panes[1-m.active]
	if passive.mode != paneFilesystem {
		m.showError(fmt.Errorf("destination panel must show a filesystem directory"))
		return m, nil
	}

	entry, ok := active.selected()
	if active.mode == paneArchive {
		return m.startOperation("Extracting archive...", func() Result {
			return extract(active.archivePath, passive.path)
		})
	}
	if !ok || entry.Name == ".." {
		m.showError(fmt.Errorf("select a file, directory, or archive first"))
		return m, nil
	}
	if entry.IsArchive && !entry.IsDir {
		return m.startOperation("Extracting archive...", func() Result {
			return extract(entry.Path, passive.path)
		})
	}

	m.modal = modalFormat
	m.modalTitle = "Pack selected item"
	m.pending = actionPack
	m.pendingPath = entry.Path
	m.formatCursor = indexOf(archiveFormats, defaultArchiveFormat)
	return m, nil
}

func (m model) startConvert() (tea.Model, tea.Cmd) {
	active := m.panes[m.active]
	passive := m.panes[1-m.active]
	if passive.mode != paneFilesystem {
		m.showError(fmt.Errorf("destination panel must show a filesystem directory"))
		return m, nil
	}

	path := active.archivePath
	if active.mode == paneFilesystem {
		entry, ok := active.selected()
		if !ok || entry.IsDir || !entry.IsArchive {
			m.showError(fmt.Errorf("select an archive to convert"))
			return m, nil
		}
		path = entry.Path
	}
	if path == "" {
		m.showError(fmt.Errorf("select an archive to convert"))
		return m, nil
	}

	m.modal = modalFormat
	m.modalTitle = "Convert archive"
	m.pending = actionConvert
	m.pendingPath = path
	m.formatCursor = indexOf(archiveFormats, defaultArchiveFormat)
	return m, nil
}

func (m model) runPending(action pendingAction, source, format string) (tea.Model, tea.Cmd) {
	destination := m.panes[1-m.active].path
	switch action {
	case actionPack:
		base := defaultOutputName(source)
		name := availableArchiveName(destination, base, format)
		return m.startOperation("Creating archive...", func() Result {
			return compress(format, name, source, destination, 3)
		})
	case actionConvert:
		base := defaultOutputName(source)
		name := availableArchiveName(destination, base, format)
		dest := filepath.Join(destination, name+"."+format)
		return m.startOperation("Converting archive...", func() Result {
			return convert(source, dest)
		})
	default:
		return m, nil
	}
}

func (m model) startOperation(status string, fn func() Result) (tea.Model, tea.Cmd) {
	m.busy = true
	m.status = status
	return m, func() tea.Msg {
		return operationMsg{result: fn()}
	}
}

func (m *model) reloadPanes() {
	for i := range m.panes {
		if err := m.panes[i].reload(); err != nil {
			m.panes[i].err = err.Error()
		}
	}
	m.ensureVisible()
}

func (m *model) ensureVisible() {
	rows := m.panelRows()
	for i := range m.panes {
		m.panes[i].ensureVisible(rows)
	}
}

func (m model) panelRows() int {
	rows := m.height - 9
	if rows < 5 {
		return 5
	}
	return rows
}
