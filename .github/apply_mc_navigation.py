from pathlib import Path


def replace(path: str, old: str, new: str) -> None:
    file = Path(path)
    text = file.read_text()
    if old not in text:
        raise SystemExit(f"pattern not found in {path}: {old[:80]!r}")
    file.write_text(text.replace(old, new, 1))


replace(
    "arx-go/main.go",
    "\tconfirm        confirmKind\n\tconfirmArchive string\n\tconfirmEntries []fileEntry\n\n\tlastClickPanel int",
    "\tconfirm        confirmKind\n\tconfirmArchive string\n\tconfirmEntries []fileEntry\n\n\tnavMenuCursor int\n\tnavInputKind  navigationInputKind\n\tnavInputValue string\n\tnavListKind   navigationListKind\n\tnavListCursor int\n\tnavListItems  []navigationItem\n\tquickSearch   string\n\thistory       [2][]paneLocation\n\thistoryIndex  [2]int\n\n\tlastClickPanel int",
)

replace(
    "arx-go/main.go",
    "\tcase modalConfirm:\n\t\treturn m.updateConfirm(msg)\n\tdefault:",
    "\tcase modalConfirm:\n\t\treturn m.updateConfirm(msg)\n\tcase modalNavigationMenu:\n\t\treturn m.updateNavigationMenu(msg)\n\tcase modalNavigationInput:\n\t\treturn m.updateNavigationInput(msg)\n\tcase modalNavigationList:\n\t\treturn m.updateNavigationList(msg)\n\tdefault:",
)

replace(
    "arx-go/main.go",
    "\tcase \"f9\", \".\":\n\t\tactive.showHidden = !active.showHidden\n\t\tif err := active.reload(); err != nil {\n\t\t\tm.showError(err)\n\t\t} else if active.showHidden {\n\t\t\tm.status = \"Hidden files: shown\"\n\t\t} else {\n\t\t\tm.status = \"Hidden files: hidden\"\n\t\t}\n\tcase \"ctrl+r\", \"r\":",
    "\tcase \"f9\":\n\t\treturn m.openNavigationMenu(), nil\n\tcase \"ctrl+h\", \".\":\n\t\tm.toggleHidden()\n\tcase \"ctrl+l\", \"alt+c\":\n\t\treturn m.openNavigationInput(navigationInputPath, \"Change directory\", active.location()), nil\n\tcase \"ctrl+s\", \"alt+s\", \"/\":\n\t\tif m.quickSearch != \"\" && msg.String() == \"ctrl+s\" {\n\t\t\tm.searchNext()\n\t\t} else {\n\t\t\treturn m.openNavigationInput(navigationInputSearch, \"Quick search\", m.quickSearch), nil\n\t\t}\n\tcase \"alt+y\":\n\t\tm.historyBack()\n\tcase \"alt+u\":\n\t\tm.historyForward()\n\tcase \"alt+h\":\n\t\treturn m.openHistoryList(), nil\n\tcase \"ctrl+\\\\\":\n\t\treturn m.openFavoritesList(), nil\n\tcase \"ctrl+b\":\n\t\tm.addCurrentFavorite()\n\tcase \"ctrl+r\", \"r\":",
)

replace(
    "arx-go/main.go",
    "\tcase \"enter\", \"right\":\n\t\tif err := active.openSelected(); err != nil {",
    "\tcase \"enter\", \"right\":\n\t\tbefore := m.currentPaneLocation(m.active)\n\t\tif err := active.openSelected(); err != nil {",
)
replace(
    "arx-go/main.go",
    "\t\t} else {\n\t\t\tm.status = active.location()\n\t\t}\n\tcase \"left\", \"backspace\":",
    "\t\t} else {\n\t\t\tm.recordNavigation(m.active, before)\n\t\t\tm.status = active.location()\n\t\t}\n\tcase \"left\", \"backspace\":",
)
replace(
    "arx-go/main.go",
    "\tcase \"left\", \"backspace\":\n\t\tif err := active.goUp(); err != nil {",
    "\tcase \"left\", \"backspace\":\n\t\tbefore := m.currentPaneLocation(m.active)\n\t\tif err := active.goUp(); err != nil {",
)
replace(
    "arx-go/main.go",
    "\t\t} else {\n\t\t\tm.status = active.location()\n\t\t}\n\tcase \"f1\":",
    "\t\t} else {\n\t\t\tm.recordNavigation(m.active, before)\n\t\t\tm.status = active.location()\n\t\t}\n\tcase \"f1\":",
)

replace(
    "arx-go/main.go",
    "\t\tm.confirmArchive = \"\"\n\t\tm.confirmEntries = nil\n}",
    "\t\tm.confirmArchive = \"\"\n\t\tm.confirmEntries = nil\n\t\tm.navInputKind = navigationInputNone\n\t\tm.navInputValue = \"\"\n\t\tm.navListKind = navigationListNone\n\t\tm.navListCursor = 0\n\t\tm.navListItems = nil\n}",
)

replace(
    "arx-go/view.go",
    "\t\t{\"F9\", \"Hidden\"},",
    "\t\t{\"F9\", \"Menu\"},",
)
replace(
    "arx-go/view.go",
    "\tcase modalConfirm:\n\t\tbody.WriteString(m.modalMessage)",
    "\tcase modalConfirm:\n\t\tbody.WriteString(m.modalMessage)",
)
replace(
    "arx-go/view.go",
    "\tcase modalConfirm:\n\t\tbody.WriteString(m.modalMessage)\n\t\tbody.WriteString(\"\\n\\n\")\n\t\tbody.WriteString(errorStyle.Render(\"Enter/Y confirms\"))\n\t\tbody.WriteString(\"   \" )\n\t\tbody.WriteString(mutedStyle.Render(\"Esc/N cancels\"))\n\t}",
    "\tcase modalConfirm:\n\t\tbody.WriteString(m.modalMessage)\n\t\tbody.WriteString(\"\\n\\n\")\n\t\tbody.WriteString(errorStyle.Render(\"Enter/Y confirms\"))\n\t\tbody.WriteString(\"   \" )\n\t\tbody.WriteString(mutedStyle.Render(\"Esc/N cancels\"))\n\tcase modalNavigationMenu, modalNavigationInput, modalNavigationList:\n\t\tbody.WriteString(m.renderNavigationModal())\n\t}",
)

# The exact spacing above can differ after gofmt; use a second targeted replacement.
view = Path("arx-go/view.go")
text = view.read_text()
if "case modalNavigationMenu" not in text:
    old = "\tcase modalConfirm:\n\t\tbody.WriteString(m.modalMessage)\n\t\tbody.WriteString(\"\\n\\n\")\n\t\tbody.WriteString(errorStyle.Render(\"Enter/Y confirms\"))\n\t\tbody.WriteString(\"   \" )\n\t\tbody.WriteString(mutedStyle.Render(\"Esc/N cancels\"))\n\t}"
    old = old.replace('\"   \" )', '\"   \")')
    new = old[:-2] + "\tcase modalNavigationMenu, modalNavigationInput, modalNavigationList:\n\t\tbody.WriteString(m.renderNavigationModal())\n\t}\n"
    if old not in text:
        raise SystemExit("navigation modal insertion point not found")
    view.write_text(text.replace(old, new, 1))

replace(
    "arx-go/view.go",
    "\t\tbody.WriteString(\"F9 or .         show/hide dot files\\n\")\n\t\tbody.WriteString(\"Ctrl-R          refresh panels\\n\")",
    "\t\tbody.WriteString(\"F9              open command menu\\n\")\n\t\tbody.WriteString(\"Ctrl-H or .      show/hide dot files\\n\")\n\t\tbody.WriteString(\"Ctrl-L / Alt-C   change directory\\n\")\n\t\tbody.WriteString(\"Ctrl-S / Alt-S   quick search; Ctrl-S repeats\\n\")\n\t\tbody.WriteString(\"Alt-Y / Alt-U    history back / forward\\n\")\n\t\tbody.WriteString(\"Alt-H            directory history\\n\")\n\t\tbody.WriteString(\"Ctrl-\\\\          favorites; Ctrl-B adds current location\\n\")\n\t\tbody.WriteString(\"Ctrl-R          refresh panels\\n\")",
)

Path("arx-go/navigation.go").write_text(r'''package main

import (
    "bufio"
    "fmt"
    "os"
    "path/filepath"
    "strings"

    tea "github.com/charmbracelet/bubbletea"
)

const (
    modalNavigationMenu modalKind = 100 + iota
    modalNavigationInput
    modalNavigationList
)

type navigationInputKind int

const (
    navigationInputNone navigationInputKind = iota
    navigationInputPath
    navigationInputSearch
)

type navigationListKind int

const (
    navigationListNone navigationListKind = iota
    navigationListHistory
    navigationListFavorites
)

type paneLocation struct {
    mode          paneMode
    path          string
    archivePath   string
    archivePrefix string
}

type navigationItem struct {
    label    string
    location string
}

var navigationMenuItems = []string{
    "Change directory",
    "Quick search",
    "Directory history",
    "Favorites",
    "Add current location to favorites",
    "Show/hide hidden files",
    "Refresh panels",
}

func (m model) openNavigationMenu() model {
    m.modal = modalNavigationMenu
    m.modalTitle = "Command menu"
    m.navMenuCursor = 0
    return m
}

func (m model) openNavigationInput(kind navigationInputKind, title, value string) model {
    m.modal = modalNavigationInput
    m.modalTitle = title
    m.navInputKind = kind
    m.navInputValue = value
    return m
}

func (m model) updateNavigationMenu(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
    switch msg.String() {
    case "esc", "f9", "q":
        m.closeModal()
    case "up", "k":
        m.navMenuCursor = wrapIndex(m.navMenuCursor-1, len(navigationMenuItems))
    case "down", "j":
        m.navMenuCursor = wrapIndex(m.navMenuCursor+1, len(navigationMenuItems))
    case "enter", "right":
        selection := m.navMenuCursor
        m.closeModal()
        switch selection {
        case 0:
            return m.openNavigationInput(navigationInputPath, "Change directory", m.panes[m.active].location()), nil
        case 1:
            return m.openNavigationInput(navigationInputSearch, "Quick search", m.quickSearch), nil
        case 2:
            return m.openHistoryList(), nil
        case 3:
            return m.openFavoritesList(), nil
        case 4:
            m.addCurrentFavorite()
        case 5:
            m.toggleHidden()
        case 6:
            m.reloadPanes()
            m.status = "Panels refreshed"
        }
    }
    return m, nil
}

func (m model) updateNavigationInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
    switch msg.String() {
    case "esc":
        m.closeModal()
        return m, nil
    case "enter":
        value := strings.TrimSpace(m.navInputValue)
        kind := m.navInputKind
        m.closeModal()
        if value == "" {
            return m, nil
        }
        if kind == navigationInputSearch {
            m.quickSearch = value
            m.searchFrom(m.panes[m.active].cursor + 1)
            return m, nil
        }
        if err := m.openLocation(value); err != nil {
            m.showError(err)
        }
        return m, nil
    case "backspace":
        runes := []rune(m.navInputValue)
        if len(runes) > 0 {
            m.navInputValue = string(runes[:len(runes)-1])
        }
        return m, nil
    case "ctrl+u":
        m.navInputValue = ""
        return m, nil
    }
    if msg.Type == tea.KeyRunes {
        m.navInputValue += string(msg.Runes)
    }
    return m, nil
}

func (m model) updateNavigationList(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
    switch msg.String() {
    case "esc", "q":
        m.closeModal()
    case "up", "k":
        m.navListCursor = wrapIndex(m.navListCursor-1, len(m.navListItems))
    case "down", "j":
        m.navListCursor = wrapIndex(m.navListCursor+1, len(m.navListItems))
    case "enter", "right":
        if len(m.navListItems) == 0 {
            m.closeModal()
            return m, nil
        }
        location := m.navListItems[m.navListCursor].location
        m.closeModal()
        if err := m.openLocation(location); err != nil {
            m.showError(err)
        }
    }
    return m, nil
}

func (m model) renderNavigationModal() string {
    switch m.modal {
    case modalNavigationMenu:
        var b strings.Builder
        for index, item := range navigationMenuItems {
            marker := "  "
            if index == m.navMenuCursor {
                marker = "> "
            }
            b.WriteString(marker + item + "\n")
        }
        b.WriteString("\n" + mutedStyle.Render("Arrows select · Enter opens · Esc closes"))
        return b.String()
    case modalNavigationInput:
        value := m.navInputValue
        if value == "" {
            value = " "
        }
        return activeFieldStyle.Width(50).Render(value) + "\n\n" + mutedStyle.Render("Enter confirms · Ctrl-U clears · Esc cancels")
    case modalNavigationList:
        if len(m.navListItems) == 0 {
            return mutedStyle.Render("No entries")
        }
        var b strings.Builder
        for index, item := range m.navListItems {
            marker := "  "
            if index == m.navListCursor {
                marker = "> "
            }
            b.WriteString(marker + item.label + "\n")
        }
        b.WriteString("\n" + mutedStyle.Render("Arrows select · Enter opens · Esc closes"))
        return b.String()
    default:
        return ""
    }
}

func (m *model) toggleHidden() {
    pane := &m.panes[m.active]
    pane.showHidden = !pane.showHidden
    if err := pane.reload(); err != nil {
        m.showError(err)
        return
    }
    if pane.showHidden {
        m.status = "Hidden files: shown"
    } else {
        m.status = "Hidden files: hidden"
    }
}

func (m *model) searchNext() {
    m.searchFrom(m.panes[m.active].cursor + 1)
}

func (m *model) searchFrom(start int) {
    pane := &m.panes[m.active]
    query := strings.ToLower(strings.TrimSpace(m.quickSearch))
    if query == "" || len(pane.entries) == 0 {
        return
    }
    for pass := 0; pass < 2; pass++ {
        from, to := start, len(pane.entries)
        if pass == 1 {
            from, to = 0, minInt(start, len(pane.entries))
        }
        for index := from; index < to; index++ {
            if strings.Contains(strings.ToLower(pane.entries[index].Name), query) {
                pane.cursor = index
                pane.ensureVisible(m.panelRows())
                m.status = "Search: " + m.quickSearch
                return
            }
        }
    }
    m.status = "No match: " + m.quickSearch
}

func (m *model) currentPaneLocation(index int) paneLocation {
    pane := m.panes[index]
    return paneLocation{mode: pane.mode, path: pane.path, archivePath: pane.archivePath, archivePrefix: pane.archivePrefix}
}

func sameLocation(left, right paneLocation) bool {
    return left.mode == right.mode && left.path == right.path && left.archivePath == right.archivePath && left.archivePrefix == right.archivePrefix
}

func (m *model) recordNavigation(index int, before paneLocation) {
    after := m.currentPaneLocation(index)
    if sameLocation(before, after) {
        return
    }
    history := m.history[index]
    position := m.historyIndex[index]
    if len(history) == 0 {
        history = append(history, before)
        position = 0
    }
    if position+1 < len(history) {
        history = history[:position+1]
    }
    history = append(history, after)
    m.history[index] = history
    m.historyIndex[index] = len(history) - 1
}

func (m *model) restoreLocation(index int, location paneLocation) error {
    pane := &m.panes[index]
    pane.clearMarks()
    if location.mode == paneArchive {
        if err := pane.loadArchive(location.archivePath); err != nil {
            return err
        }
        pane.archivePrefix = location.archivePrefix
        return pane.loadArchiveView()
    }
    pane.mode = paneFilesystem
    pane.path = location.path
    pane.archivePath = ""
    pane.archivePrefix = ""
    pane.archiveItems = nil
    pane.cursor = 0
    pane.offset = 0
    return pane.loadDirectory()
}

func (m *model) historyBack() {
    history := m.history[m.active]
    position := m.historyIndex[m.active]
    if len(history) == 0 || position <= 0 {
        m.status = "No previous location"
        return
    }
    position--
    if err := m.restoreLocation(m.active, history[position]); err != nil {
        m.showError(err)
        return
    }
    m.historyIndex[m.active] = position
    m.status = m.panes[m.active].location()
}

func (m *model) historyForward() {
    history := m.history[m.active]
    position := m.historyIndex[m.active]
    if len(history) == 0 || position+1 >= len(history) {
        m.status = "No next location"
        return
    }
    position++
    if err := m.restoreLocation(m.active, history[position]); err != nil {
        m.showError(err)
        return
    }
    m.historyIndex[m.active] = position
    m.status = m.panes[m.active].location()
}

func locationString(location paneLocation) string {
    if location.mode == paneArchive {
        return location.archivePath + ":/" + location.archivePrefix
    }
    return location.path
}

func (m model) openHistoryList() model {
    items := make([]navigationItem, 0, len(m.history[m.active]))
    for _, location := range m.history[m.active] {
        value := locationString(location)
        items = append(items, navigationItem{label: value, location: value})
    }
    m.modal = modalNavigationList
    m.modalTitle = "Directory history"
    m.navListKind = navigationListHistory
    m.navListItems = items
    m.navListCursor = maxInt(0, len(items)-1)
    return m
}

func expandLocation(value string) string {
    value = os.ExpandEnv(strings.TrimSpace(value))
    if value == "~" || strings.HasPrefix(value, "~/") {
        if home, err := os.UserHomeDir(); err == nil {
            value = filepath.Join(home, strings.TrimPrefix(value, "~/"))
        }
    }
    return value
}

func splitArchiveLocation(value string) (string, string, bool) {
    marker := ":/"
    index := strings.LastIndex(value, marker)
    if index <= 1 {
        return "", "", false
    }
    archive := value[:index]
    if DetectArchiveFormat(archive) == "unknown" {
        return "", "", false
    }
    return archive, strings.Trim(value[index+len(marker):], "/"), true
}

func (m *model) openLocation(value string) error {
    value = expandLocation(value)
    before := m.currentPaneLocation(m.active)
    pane := &m.panes[m.active]
    if archive, prefix, ok := splitArchiveLocation(value); ok {
        if err := pane.loadArchive(archive); err != nil {
            return err
        }
        pane.archivePrefix = prefix
        if err := pane.loadArchiveView(); err != nil {
            return err
        }
    } else {
        absolute, err := filepath.Abs(value)
        if err != nil {
            return err
        }
        info, err := os.Stat(absolute)
        if err != nil {
            return err
        }
        if info.IsDir() {
            pane.mode = paneFilesystem
            pane.path = absolute
            pane.archivePath = ""
            pane.archivePrefix = ""
            pane.archiveItems = nil
            pane.cursor = 0
            pane.offset = 0
            if err := pane.loadDirectory(); err != nil {
                return err
            }
        } else if info.Mode().IsRegular() && DetectArchiveFormat(absolute) != "unknown" {
            if err := pane.loadArchive(absolute); err != nil {
                return err
            }
        } else {
            return fmt.Errorf("not a directory or supported archive: %s", value)
        }
    }
    m.recordNavigation(m.active, before)
    m.status = pane.location()
    return nil
}

func favoritesPath() (string, error) {
    config, err := os.UserConfigDir()
    if err != nil {
        return "", err
    }
    return filepath.Join(config, "arx", "favorites"), nil
}

func loadFavorites() ([]navigationItem, error) {
    path, err := favoritesPath()
    if err != nil {
        return nil, err
    }
    file, err := os.Open(path)
    if os.IsNotExist(err) {
        return nil, nil
    }
    if err != nil {
        return nil, err
    }
    defer file.Close()
    var items []navigationItem
    scanner := bufio.NewScanner(file)
    for scanner.Scan() {
        value := strings.TrimSpace(scanner.Text())
        if value != "" {
            items = append(items, navigationItem{label: value, location: value})
        }
    }
    return items, scanner.Err()
}

func saveFavorite(value string) error {
    items, err := loadFavorites()
    if err != nil {
        return err
    }
    for _, item := range items {
        if item.location == value {
            return nil
        }
    }
    path, err := favoritesPath()
    if err != nil {
        return err
    }
    if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
        return err
    }
    file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
    if err != nil {
        return err
    }
    defer file.Close()
    _, err = fmt.Fprintln(file, value)
    return err
}

func (m *model) addCurrentFavorite() {
    value := locationString(m.currentPaneLocation(m.active))
    if err := saveFavorite(value); err != nil {
        m.showError(err)
        return
    }
    m.status = "Favorite added: " + value
}

func (m model) openFavoritesList() model {
    items, err := loadFavorites()
    if err != nil {
        m.showError(err)
        return m
    }
    m.modal = modalNavigationList
    m.modalTitle = "Favorites"
    m.navListKind = navigationListFavorites
    m.navListItems = items
    m.navListCursor = 0
    return m
}

func maxInt(left, right int) int {
    if left > right {
        return left
    }
    return right
}
''')

Path("arx-go/navigation_test.go").write_text(r'''package main

import (
    "os"
    "path/filepath"
    "testing"

    tea "github.com/charmbracelet/bubbletea"
)

func TestQuickSearchWrapsAndMatchesCaseInsensitive(t *testing.T) {
    dir := t.TempDir()
    for _, name := range []string{"alpha.txt", "Beta.txt", "gamma.txt"} {
        if err := os.WriteFile(filepath.Join(dir, name), []byte(name), 0o644); err != nil {
            t.Fatal(err)
        }
    }
    model := initialModelAt(dir)
    model.quickSearch = "beta"
    model.panes[0].cursor = len(model.panes[0].entries) - 1
    model.searchNext()
    entry, ok := model.panes[0].selected()
    if !ok || entry.Name != "Beta.txt" {
        t.Fatalf("selected=%q", entry.Name)
    }
}

func TestOpenLocationAndHistory(t *testing.T) {
    root := t.TempDir()
    child := filepath.Join(root, "child")
    if err := os.Mkdir(child, 0o755); err != nil {
        t.Fatal(err)
    }
    model := initialModelAt(root)
    if err := model.openLocation(child); err != nil {
        t.Fatal(err)
    }
    if model.panes[0].path != child {
        t.Fatalf("path=%q", model.panes[0].path)
    }
    model.historyBack()
    if model.panes[0].path != root {
        t.Fatalf("back path=%q", model.panes[0].path)
    }
    model.historyForward()
    if model.panes[0].path != child {
        t.Fatalf("forward path=%q", model.panes[0].path)
    }
}

func TestFavoritesPersistWithoutDuplicates(t *testing.T) {
    t.Setenv("XDG_CONFIG_HOME", t.TempDir())
    location := filepath.Join(t.TempDir(), "docs")
    if err := saveFavorite(location); err != nil {
        t.Fatal(err)
    }
    if err := saveFavorite(location); err != nil {
        t.Fatal(err)
    }
    items, err := loadFavorites()
    if err != nil {
        t.Fatal(err)
    }
    if len(items) != 1 || items[0].location != location {
        t.Fatalf("favorites=%v", items)
    }
}

func TestF9OpensMenuAndCtrlHTogglesHidden(t *testing.T) {
    model := initialModelAt(t.TempDir())
    updated, _ := model.updateBrowser(tea.KeyMsg{Type: tea.KeyF9})
    menu := updated.(model)
    if menu.modal != modalNavigationMenu {
        t.Fatalf("modal=%v", menu.modal)
    }
    menu.closeModal()
    before := menu.panes[0].showHidden
    updated, _ = menu.updateBrowser(tea.KeyMsg{Type: tea.KeyCtrlH})
    toggled := updated.(model)
    if toggled.panes[0].showHidden == before {
        t.Fatal("Ctrl-H did not toggle hidden files")
    }
}
''')

# Format and leave the repository ready for the normal test steps.
