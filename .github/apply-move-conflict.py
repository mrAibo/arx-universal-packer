from pathlib import Path


def replace_once(path: str, old: str, new: str) -> None:
    file = Path(path)
    text = file.read_text()
    if text.count(old) != 1:
        raise SystemExit(f"expected one match in {path}, got {text.count(old)}")
    file.write_text(text.replace(old, new))


replace_once(
    "arx-go/filesystem_move.go",
    '''func (m model) startFilesystemMove(entries []fileEntry, target, baseDir string) (tea.Model, tea.Cmd) {
	target, err := normalizeMoveTarget(target, baseDir)
	if err != nil {
		m.showError(err)
		return m, nil
	}
	_, conflicts, err := filesystemMovePlans(entries, target)
	if err != nil {
		m.showError(err)
		return m, nil
	}
	if len(conflicts) > 0 {
		m.modal = modalConfirm
		m.modalTitle = "Replace existing items"
		m.modalMessage = moveConflictMessage(conflicts, target)
		m.confirm = confirmFilesystemMove
		m.confirmEntries = append([]fileEntry(nil), entries...)
		m.confirmDestination = target
		return m, nil
	}
	return m.runFilesystemMove(entries, target, false)
}
''',
    '''func (m model) startFilesystemMove(entries []fileEntry, target, baseDir string) (tea.Model, tea.Cmd) {
	return m.startFilesystemMoveWithConflicts(entries, target, baseDir)
}
''',
)

replace_once(
    "arx-go/main.go",
    '''	copyPlans            []filesystemCopyPlan
	copyConflictIndex    int
	copyConflictAction   copyConflictAction
	copyConflictApplyAll bool
	copyConflictRename   string
''',
    '''	copyPlans            []filesystemCopyPlan
	copyConflictIndex    int
	copyConflictAction   copyConflictAction
	copyConflictApplyAll bool
	copyConflictRename   string

	movePlans            []filesystemMoveDecisionPlan
	moveConflictIndex    int
	moveConflictAction   copyConflictAction
	moveConflictApplyAll bool
	moveConflictRename   string
	moveConflictTarget   string
''',
)

replace_once(
    "arx-go/main.go",
    '''	case modalCopyConflict:
		return m.updateCopyConflict(msg)
	case modalNavigationMenu:
''',
    '''	case modalCopyConflict:
		return m.updateCopyConflict(msg)
	case modalMoveConflict:
		return m.updateMoveConflict(msg)
	case modalNavigationMenu:
''',
)

replace_once(
    "arx-go/view.go",
    '''	case modalCopyConflict:
		body.WriteString(m.renderCopyConflict())
	case modalNavigationMenu, modalNavigationInput, modalNavigationList:
''',
    '''	case modalCopyConflict:
		body.WriteString(m.renderCopyConflict())
	case modalMoveConflict:
		body.WriteString(m.renderMoveConflict())
	case modalNavigationMenu, modalNavigationInput, modalNavigationList:
''',
)
