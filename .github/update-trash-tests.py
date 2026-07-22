from pathlib import Path


def replace_once(path: str, old: str, new: str) -> None:
    file = Path(path)
    text = file.read_text()
    count = text.count(old)
    if count != 1:
        raise SystemExit(f"{path}: expected one match, found {count}")
    file.write_text(text.replace(old, new, 1))


replace_once(
    "arx-go/navigation_test.go",
    'if got := m.f8Label(); got != "Clear" {',
    'if got := m.f8Label(); got != "Trash" {',
)
replace_once(
    "arx-go/tui_test.go",
    "func TestCtrlAMarksAllAndF8Clears(t *testing.T) {",
    "func TestCtrlAMarksAllAndCtrlUClears(t *testing.T) {",
)
replace_once(
    "arx-go/tui_test.go",
    'updated, _ = m.Update(runeKey("f8"))',
    'updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlU})',
)
