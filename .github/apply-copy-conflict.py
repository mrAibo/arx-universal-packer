from pathlib import Path


def replace_once(path: str, old: str, new: str) -> None:
    file = Path(path)
    text = file.read_text()
    count = text.count(old)
    if count != 1:
        raise SystemExit(f"{path}: expected one match, found {count}")
    file.write_text(text.replace(old, new, 1))


replace_once(
    "arx-go/filesystem_copy.go",
    '''func (m model) startFilesystemCopy(entries []fileEntry, destination string) (tea.Model, tea.Cmd) {
	conflicts, err := filesystemCopyConflicts(entries, destination)
	if err != nil {
		m.showError(err)
		return m, nil
	}
	if len(conflicts) > 0 {
		m.modal = modalConfirm
		m.modalTitle = "Overwrite existing items"
		m.modalMessage = copyConflictMessage(conflicts, destination)
		m.confirm = confirmFilesystemCopy
		m.confirmEntries = append([]fileEntry(nil), entries...)
		m.confirmDestination = destination
		return m, nil
	}
	return m.runFilesystemCopy(entries, destination, false)
}

func (m model) runFilesystemCopy(entries []fileEntry, destination string, overwrite bool) (tea.Model, tea.Cmd) {
	items := append([]fileEntry(nil), entries...)
	return m.startOperation(fmt.Sprintf("Copying %d selected item(s)...", len(items)), func() Result {
		return copyFilesystem(items, destination, overwrite)
	})
}

''',
    '',
)

replace_once(
    "arx-go/main.go",
    '''	confirm            confirmKind
	confirmArchive     string
	confirmEntries     []fileEntry
	confirmDestination string

''',
    '''	confirm            confirmKind
	confirmArchive     string
	confirmEntries     []fileEntry
	confirmDestination string

	copyPlans            []filesystemCopyPlan
	copyConflictIndex    int
	copyConflictAction   copyConflictAction
	copyConflictApplyAll bool
	copyConflictRename   string

''',
)

replace_once(
    "arx-go/main.go",
    '''	case modalConfirm:
		return m.updateConfirm(msg)
	case modalNavigationMenu:
''',
    '''	case modalConfirm:
		return m.updateConfirm(msg)
	case modalCopyConflict:
		return m.updateCopyConflict(msg)
	case modalNavigationMenu:
''',
)

replace_once(
    "arx-go/view.go",
    '''	case modalConfirm:
		body.WriteString(m.modalMessage)
		body.WriteString("\\n\\n")
		body.WriteString(errorStyle.Render("Enter/Y confirms"))
		body.WriteString("   ")
		body.WriteString(mutedStyle.Render("Esc/N cancels"))
	case modalNavigationMenu, modalNavigationInput, modalNavigationList:
''',
    '''	case modalConfirm:
		body.WriteString(m.modalMessage)
		body.WriteString("\\n\\n")
		body.WriteString(errorStyle.Render("Enter/Y confirms"))
		body.WriteString("   ")
		body.WriteString(mutedStyle.Render("Esc/N cancels"))
	case modalCopyConflict:
		body.WriteString(m.renderCopyConflict())
	case modalNavigationMenu, modalNavigationInput, modalNavigationList:
''',
)
