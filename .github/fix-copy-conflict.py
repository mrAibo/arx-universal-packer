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
    '\n\ttea "github.com/charmbracelet/bubbletea"\n',
    '',
)

replace_once(
    "arx-go/filesystem_copy_test.go",
    '''	confirmation := updated.(model)
	if confirmation.modal != modalConfirm || confirmation.confirm != confirmFilesystemCopy {
		t.Fatalf("modal=%v confirm=%v", confirmation.modal, confirmation.confirm)
	}

	updated, command = confirmation.updateConfirm(tea.KeyMsg{Type: tea.KeyEnter})
''',
    '''	confirmation := updated.(model)
	if confirmation.modal != modalCopyConflict || confirmation.copyConflictAction != copyConflictReplace {
		t.Fatalf("modal=%v action=%v", confirmation.modal, confirmation.copyConflictAction)
	}

	updated, command = confirmation.updateCopyConflict(tea.KeyMsg{Type: tea.KeyEnter})
''',
)
