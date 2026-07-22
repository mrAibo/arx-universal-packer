package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	defaultArchiveFormat = "tar.zst"
	defaultLevel         = 3
	doubleClickWindow    = 450 * time.Millisecond
)

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

	markedStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("58")).
			Foreground(lipgloss.Color("229")).
			Bold(true)

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

	fieldStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240")).
			Padding(0, 1)

	activeFieldStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("39")).
				Padding(0, 1)
)

type modalKind int

const (
	modalNone modalKind = iota
	modalHelp
	modalArchive
	modalMessage
	modalViewer
	modalConfirm
)

type pendingAction int

const (
	actionNone pendingAction = iota
	actionPack
	actionConvert
)

const (
	dialogName = iota
	dialogFormat
	dialogLevel
	dialogCreate
	dialogFieldCount
)

type operationMsg struct {
	result Result
}

type model struct {
	panes  [2]pane
	active int
	width  int
	height int
	status string
	busy   bool

	modal        modalKind
	modalTitle   string
	modalMessage string

	pending         pendingAction
	pendingSources  []string
	pendingBaseDir  string
	archiveName     string
	formatCursor    int
	compression     int
	dialogField     int
	dialogError     string
	nameReplaceMode bool

	viewerLines  []string
	viewerOffset int
	viewerColumn int

	confirm            confirmKind
	confirmArchive     string
	confirmEntries     []fileEntry
	confirmDestination string

	navMenuCursor int
	navInputKind  navigationInputKind
	navInputValue string
	navListKind   navigationListKind
	navListCursor int
	navListItems  []navigationItem
	quickSearch   string
	history       [2][]paneLocation
	historyIndex  [2]int

	lastClickPanel int
	lastClickIndex int
	lastClickAt    time.Time
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
		panes:          [2]pane{left, right},
		active:         0,
		width:          100,
		height:         30,
		status:         "Ready",
		compression:    defaultLevel,
		lastClickPanel: -1,
		lastClickIndex: -1,
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
			for i := range m.panes {
				m.panes[i].clearMarks()
			}
			m.reloadPanes()
		}
		return m, nil
	case tea.MouseMsg:
		if m.modal == modalViewer {
			return m.updateViewerMouse(msg)
		}
		if m.modal != modalNone || m.busy {
			return m, nil
		}
		return m.updateMouse(msg)
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
		m.status = m.panes[m.active].location()
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
	case " ", "space", "insert":
		active.toggleMark(rows)
		m.status = active.markSummary()
	case "f2":
		return m.startPack()
	case "ctrl+a":
		active.markAll()
		m.status = active.markSummary()
	case "*":
		active.invertMarks()
		m.status = active.markSummary()
	case "ctrl+u":
		active.clearMarks()
		m.status = "Marks cleared"
	case "enter", "right":
		before := m.currentPaneLocation(m.active)
		if err := active.openSelected(); err != nil {
			m.showError(err)
		} else {
			m.recordNavigation(m.active, before)
			m.status = active.location()
		}
	case "left", "backspace":
		before := m.currentPaneLocation(m.active)
		if err := active.goUp(); err != nil {
			m.showError(err)
		} else {
			m.recordNavigation(m.active, before)
			m.status = active.location()
		}
	case "f1":
		m.modal = modalHelp
		m.modalTitle = "ARX Commander help"
	case "f3":
		return m.startViewer()
	case "f4":
		return m.startArchiveTest()
	case "f5":
		return m.startF5()
	case "f6":
		return m.startMove()
	case "alt+f6":
		return m.startConvert()
	case "f7":
		if active.mode != paneFilesystem {
			m.showError(fmt.Errorf("cannot create a directory inside an archive yet"))
			return m, nil
		}
		return m.openNavigationInput(navigationInputMkdir, "Create directory", ""), nil
	case "f8":
		return m.startFilesystemTrash()
	case "f9":
		return m.openNavigationMenu(), nil
	case "ctrl+h", ".":
		m.toggleHidden()
	case "ctrl+l", "alt+c":
		return m.openNavigationInput(navigationInputPath, "Change directory", active.location()), nil
	case "ctrl+s", "alt+s", "/":
		if m.quickSearch != "" && msg.String() == "ctrl+s" {
			m.searchNext()
		} else {
			return m.openNavigationInput(navigationInputSearch, "Quick search", m.quickSearch), nil
		}
	case "alt+y":
		m.historyBack()
	case "alt+u":
		m.historyForward()
	case "alt+h":
		return m.openHistoryList(), nil
	case "ctrl+\\":
		return m.openFavoritesList(), nil
	case "ctrl+b":
		m.addCurrentFavorite()
	case "ctrl+r", "r":
		m.reloadPanes()
		m.status = "Panels refreshed"
	}

	m.ensureVisible()
	return m, nil
}

func (m model) updateMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	panel := m.panelAtX(msg.X)
	if panel < 0 {
		return m, nil
	}
	rows := m.panelRows()

	switch msg.Button {
	case tea.MouseButtonWheelUp:
		m.active = panel
		m.panes[panel].move(-3, rows)
		return m, nil
	case tea.MouseButtonWheelDown:
		m.active = panel
		m.panes[panel].move(3, rows)
		return m, nil
	}

	if msg.Action != tea.MouseActionPress {
		return m, nil
	}
	index, ok := m.entryIndexAtY(panel, msg.Y)
	if !ok {
		m.active = panel
		return m, nil
	}

	m.active = panel
	active := &m.panes[panel]
	active.selectIndex(index, rows)

	switch msg.Button {
	case tea.MouseButtonRight, tea.MouseButtonMiddle:
		active.toggleMarkIndex(index, rows, false)
		m.status = active.markSummary()
	case tea.MouseButtonLeft:
		now := time.Now()
		doubleClick := m.lastClickPanel == panel && m.lastClickIndex == index && now.Sub(m.lastClickAt) <= doubleClickWindow
		m.lastClickPanel = panel
		m.lastClickIndex = index
		m.lastClickAt = now
		if doubleClick {
			if err := active.openSelected(); err != nil {
				m.showError(err)
			} else {
				m.status = active.location()
			}
		} else {
			m.status = active.selectedDescription()
		}
	}
	return m, nil
}

func (m model) panelAtX(x int) int {
	width := m.width
	if width < 60 {
		width = 60
	}
	if x < 0 || x >= width {
		return -1
	}
	leftWidth := (width - 1) / 2
	if x < leftWidth {
		return 0
	}
	return 1
}

func (m model) entryIndexAtY(panel, y int) (int, bool) {
	row := y - 4
	if row < 0 || row >= m.panelRows() {
		return 0, false
	}
	index := m.panes[panel].offset + row
	if index < 0 || index >= len(m.panes[panel].entries) {
		return 0, false
	}
	return index, true
}

func (m model) updateModal(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.modal {
	case modalHelp, modalMessage:
		switch msg.String() {
		case "esc", "enter", "q", "f1":
			m.closeModal()
		}
		return m, nil
	case modalArchive:
		return m.updateArchiveDialog(msg)
	case modalViewer:
		return m.updateViewer(msg)
	case modalConfirm:
		return m.updateConfirm(msg)
	case modalNavigationMenu:
		return m.updateNavigationMenu(msg)
	case modalNavigationInput:
		return m.updateNavigationInput(msg)
	case modalNavigationList:
		return m.updateNavigationList(msg)
	default:
		return m, nil
	}
}

func (m model) updateArchiveDialog(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	m.dialogError = ""
	key := msg.String()

	switch key {
	case "esc":
		m.closeModal()
		return m, nil
	case "tab":
		m.nameReplaceMode = false
		m.dialogField = (m.dialogField + 1) % dialogFieldCount
		return m, nil
	case "shift+tab":
		m.nameReplaceMode = false
		m.dialogField = wrapIndex(m.dialogField-1, dialogFieldCount)
		return m, nil
	case "f5":
		return m.confirmArchiveDialog()
	case "enter":
		if m.dialogField == dialogCreate {
			return m.confirmArchiveDialog()
		}
		m.nameReplaceMode = false
		m.dialogField++
		return m, nil
	case "backspace":
		if m.dialogField == dialogName {
			if m.nameReplaceMode {
				m.archiveName = ""
				m.nameReplaceMode = false
			} else {
				runes := []rune(m.archiveName)
				if len(runes) > 0 {
					m.archiveName = string(runes[:len(runes)-1])
				}
			}
		}
		return m, nil
	case "ctrl+u":
		if m.dialogField == dialogName {
			m.archiveName = ""
			m.nameReplaceMode = false
		}
		return m, nil
	case "up", "left":
		switch m.dialogField {
		case dialogFormat:
			m.formatCursor = wrapIndex(m.formatCursor-1, len(archiveFormats))
		case dialogLevel:
			if m.compression > 1 {
				m.compression--
			}
		}
		return m, nil
	case "down", "right":
		switch m.dialogField {
		case dialogFormat:
			m.formatCursor = wrapIndex(m.formatCursor+1, len(archiveFormats))
		case dialogLevel:
			if m.compression < 9 {
				m.compression++
			}
		}
		return m, nil
	}

	if msg.Type == tea.KeyRunes && m.dialogField == dialogName {
		value := string(msg.Runes)
		if m.nameReplaceMode {
			m.archiveName = value
			m.nameReplaceMode = false
		} else {
			m.archiveName += value
		}
	}
	return m, nil
}

func (m model) confirmArchiveDialog() (tea.Model, tea.Cmd) {
	if len(m.pendingSources) == 0 {
		m.dialogError = "No source selected"
		return m, nil
	}
	format := archiveFormats[m.formatCursor]
	name, err := normalizeArchiveName(m.archiveName, format)
	if err != nil {
		m.dialogError = err.Error()
		m.dialogField = dialogName
		return m, nil
	}
	destination := m.panes[1-m.active].path
	output := filepath.Join(destination, name+"."+format)
	if _, err := os.Stat(output); err == nil {
		m.dialogError = "Archive already exists; choose another name"
		m.dialogField = dialogName
		return m, nil
	} else if !os.IsNotExist(err) {
		m.dialogError = err.Error()
		return m, nil
	}

	action := m.pending
	sources := append([]string(nil), m.pendingSources...)
	baseDir := m.pendingBaseDir
	level := m.compression
	m.closeModal()

	switch action {
	case actionPack:
		return m.startOperation("Creating archive...", func() Result {
			return compressMany(format, name, sources, baseDir, destination, level)
		})
	case actionConvert:
		return m.startOperation("Converting archive...", func() Result {
			return convertArchive(sources[0], output, level)
		})
	default:
		return m, nil
	}
}

func (m *model) closeModal() {
	m.modal = modalNone
	m.modalTitle = ""
	m.modalMessage = ""
	m.pending = actionNone
	m.pendingSources = nil
	m.pendingBaseDir = ""
	m.archiveName = ""
	m.dialogError = ""
	m.dialogField = dialogName
	m.nameReplaceMode = false
	m.viewerLines = nil
	m.viewerOffset = 0
	m.viewerColumn = 0
	m.confirm = confirmNone
	m.confirmArchive = ""
	m.confirmEntries = nil
	m.confirmDestination = ""
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

	entries, err := active.operationEntries()
	if err != nil {
		m.showError(err)
		return m, nil
	}

	if active.mode == paneArchive {
		if passive.mode != paneFilesystem {
			m.showError(fmt.Errorf("destination panel must show a filesystem directory"))
			return m, nil
		}
		members := archiveMembersForEntries(active.archiveItems, entries)
		if len(members) == 0 {
			m.showError(fmt.Errorf("selected archive entries could not be resolved"))
			return m, nil
		}
		return m.startOperation(fmt.Sprintf("Extracting %d selected item(s)...", len(entries)), func() Result {
			return extractSelected(active.archivePath, members, passive.path)
		})
	}

	if passive.mode == paneArchive {
		sources := make([]string, 0, len(entries))
		for _, entry := range entries {
			sources = append(sources, entry.Path)
		}
		return m.startOperation(fmt.Sprintf("Adding %d selected item(s) to archive...", len(entries)), func() Result {
			return addToArchive(passive.archivePath, passive.archivePrefix, sources, active.path, defaultLevel)
		})
	}
	if passive.mode != paneFilesystem {
		m.showError(fmt.Errorf("destination panel must show a filesystem directory or an opened archive"))
		return m, nil
	}

	marked := active.markedEntries()
	if len(marked) == 0 && len(entries) == 1 && entries[0].IsArchive && !entries[0].IsDir {
		archivePath := entries[0].Path
		return m.startOperation("Extracting archive...", func() Result {
			return extract(archivePath, passive.path)
		})
	}

	return m.startFilesystemCopy(entries, passive.path)
}

func (m model) startPack() (tea.Model, tea.Cmd) {
	active := m.panes[m.active]
	passive := m.panes[1-m.active]
	if active.mode != paneFilesystem || passive.mode != paneFilesystem {
		m.showError(fmt.Errorf("archive creation requires filesystem panels"))
		return m, nil
	}
	entries, err := active.operationEntries()
	if err != nil {
		m.showError(err)
		return m, nil
	}
	sources := make([]string, 0, len(entries))
	for _, entry := range entries {
		sources = append(sources, entry.Path)
	}
	name := "archive"
	if len(entries) == 1 {
		name = defaultOutputName(entries[0].Path)
	}
	return m.openArchiveDialog(actionPack, "Create archive", name, sources, active.path), nil
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
		entries, err := active.operationEntries()
		if err != nil || len(entries) != 1 || entries[0].IsDir || !entries[0].IsArchive {
			m.showError(fmt.Errorf("select exactly one archive to convert"))
			return m, nil
		}
		path = entries[0].Path
	}
	if path == "" {
		m.showError(fmt.Errorf("select an archive to convert"))
		return m, nil
	}

	return m.openArchiveDialog(actionConvert, "Convert archive", defaultOutputName(path), []string{path}, filepath.Dir(path)), nil
}

func (m model) openArchiveDialog(action pendingAction, title, name string, sources []string, baseDir string) model {
	m.modal = modalArchive
	m.modalTitle = title
	m.pending = action
	m.pendingSources = append([]string(nil), sources...)
	m.pendingBaseDir = baseDir
	m.archiveName = name
	m.formatCursor = indexOf(archiveFormats, defaultArchiveFormat)
	m.compression = defaultLevel
	m.dialogField = dialogName
	m.dialogError = ""
	m.nameReplaceMode = true
	return m
}

func (m model) startArchiveTest() (tea.Model, tea.Cmd) {
	active := m.panes[m.active]
	path := active.archivePath
	if active.mode == paneFilesystem {
		entries, err := active.operationEntries()
		if err != nil || len(entries) != 1 || entries[0].IsDir || !entries[0].IsArchive {
			m.showError(fmt.Errorf("select exactly one archive to test"))
			return m, nil
		}
		path = entries[0].Path
	}
	return m.startOperation("Testing archive...", func() Result {
		return testArchive(path)
	})
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
		if err := m.panes[i].refresh(); err != nil {
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

func normalizeArchiveName(value, format string) (string, error) {
	value = strings.TrimSpace(value)
	lower := strings.ToLower(value)
	for _, suffix := range []string{
		".tar.gz", ".tar.bz2", ".tar.xz", ".tar.zst", ".tgz", ".tbz2", ".txz", ".zip", ".7z", ".tar",
	} {
		if strings.HasSuffix(lower, suffix) {
			value = value[:len(value)-len(suffix)]
			break
		}
	}
	value = strings.TrimSpace(value)
	if value == "" || value == "." || value == ".." {
		return "", fmt.Errorf("enter an archive name")
	}
	if strings.ContainsAny(value, `/\`) || strings.ContainsRune(value, '\x00') {
		return "", fmt.Errorf("archive name must not contain path separators")
	}
	if strings.TrimSpace(format) == "" {
		return "", fmt.Errorf("select an archive format")
	}
	return value, nil
}
