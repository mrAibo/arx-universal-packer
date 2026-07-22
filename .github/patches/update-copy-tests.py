#!/usr/bin/env python3
from pathlib import Path

path = Path("arx-go/tui_test.go")
text = path.read_text(encoding="utf-8")
replacements = [
    (
        '''func TestF2MarksAllAndF8Clears(t *testing.T) {''',
        '''func TestCtrlAMarksAllAndF8Clears(t *testing.T) {''',
    ),
    (
        '''\tupdated, _ := m.Update(runeKey("f2"))\n\tm = updated.(model)\n\tif got := len(m.panes[0].markedEntries()); got != 2 {''',
        '''\tupdated, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlA})\n\tm = updated.(model)\n\tif got := len(m.panes[0].markedEntries()); got != 2 {''',
    ),
    (
        '''func TestF5WithMarkedItemsOpensNamedArchiveDialog(t *testing.T) {''',
        '''func TestF2WithMarkedItemsOpensNamedArchiveDialog(t *testing.T) {''',
    ),
    (
        '''\tm.panes[0].markAll()\n\tupdated, _ := m.Update(runeKey("f5"))''',
        '''\tm.panes[0].markAll()\n\tupdated, _ := m.Update(runeKey("f2"))''',
    ),
    (
        '''\tm.panes[0].selectName("data.txt")\n\tupdated, _ := m.Update(runeKey("f5"))''',
        '''\tm.panes[0].selectName("data.txt")\n\tupdated, _ := m.Update(runeKey("f2"))''',
    ),
]
for old, new in replacements:
    count = text.count(old)
    if count != 1:
        raise SystemExit(f"expected one replacement, found {count}: {old.splitlines()[0]}")
    text = text.replace(old, new, 1)
path.write_text(text, encoding="utf-8")
