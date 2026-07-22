#!/usr/bin/env python3
from pathlib import Path


def replace_once(path: str, old: str, new: str) -> None:
    file = Path(path)
    text = file.read_text(encoding="utf-8")
    count = text.count(old)
    if count != 1:
        raise SystemExit(f"{path}: expected one replacement, found {count}: {old.splitlines()[0]}")
    file.write_text(text.replace(old, new, 1), encoding="utf-8")


replace_once(
    "arx-go/navigation.go",
    '''\tnavigationInputPath\n\tnavigationInputSearch\n\tnavigationInputMkdir\n)''',
    '''\tnavigationInputPath\n\tnavigationInputSearch\n\tnavigationInputMkdir\n\tnavigationInputMove\n)''',
)

replace_once(
    "arx-go/navigation.go",
    '''\t"Show/hide hidden files",\n\t"Refresh panels",\n}''',
    '''\t"Show/hide hidden files",\n\t"Refresh panels",\n\t"Convert selected archive",\n}''',
)

replace_once(
    "arx-go/navigation.go",
    '''\t\tcase 6:\n\t\t\tm.reloadPanes()\n\t\t\tm.status = "Panels refreshed"\n\t\t}\n''',
    '''\t\tcase 6:\n\t\t\tm.reloadPanes()\n\t\t\tm.status = "Panels refreshed"\n\t\tcase 7:\n\t\t\treturn m.startConvert()\n\t\t}\n''',
)

replace_once(
    "arx-go/navigation.go",
    '''\t\tvalue := strings.TrimSpace(m.navInputValue)\n\t\tkind := m.navInputKind\n\t\tm.closeModal()\n\t\tif kind == navigationInputMkdir {''',
    '''\t\tvalue := strings.TrimSpace(m.navInputValue)\n\t\tkind := m.navInputKind\n\t\tsources := append([]string(nil), m.pendingSources...)\n\t\tbaseDir := m.pendingBaseDir\n\t\tm.closeModal()\n\t\tif kind == navigationInputMove {\n\t\t\tentries := make([]fileEntry, 0, len(sources))\n\t\t\tfor _, source := range sources {\n\t\t\t\tentries = append(entries, fileEntry{Path: source})\n\t\t\t}\n\t\t\treturn m.startFilesystemMove(entries, value, baseDir)\n\t\t}\n\t\tif kind == navigationInputMkdir {''',
)

replace_once(
    "arx-go/main.go",
    '''\tcase "f6":\n\t\treturn m.startConvert()\n\tcase "f7":''',
    '''\tcase "f6":\n\t\treturn m.startMove()\n\tcase "alt+f6":\n\t\treturn m.startConvert()\n\tcase "f7":''',
)

replace_once(
    "arx-go/archive_edit.go",
    '''\tconfirmArchiveDelete\n\tconfirmFilesystemCopy\n)''',
    '''\tconfirmArchiveDelete\n\tconfirmFilesystemCopy\n\tconfirmFilesystemMove\n)''',
)

replace_once(
    "arx-go/archive_edit.go",
    '''\t\tcase confirmFilesystemCopy:\n\t\t\treturn m.runFilesystemCopy(entries, destination, true)\n\t\tdefault:''',
    '''\t\tcase confirmFilesystemCopy:\n\t\t\treturn m.runFilesystemCopy(entries, destination, true)\n\t\tcase confirmFilesystemMove:\n\t\t\treturn m.runFilesystemMove(entries, destination, true)\n\t\tdefault:''',
)

replace_once(
    "arx-go/view.go",
    '''\t\t{"F5", m.f5Label()},\n\t\t{"F6", "Conv"},''',
    '''\t\t{"F5", m.f5Label()},\n\t\t{"F6", "Move"},''',
)

replace_once(
    "arx-go/view.go",
    '''\t\tbody.WriteString("F5              copy/extract/add according to panel direction\\n")\n\t\tbody.WriteString("F6              convert selected archive\\n")''',
    '''\t\tbody.WriteString("F5              copy/extract/add according to panel direction\\n")\n\t\tbody.WriteString("F6              move/rename selected filesystem items\\n")\n\t\tbody.WriteString("Alt-F6          convert selected archive\\n")''',
)

replace_once(
    "arx-go/view.go",
    '''\t\tbody.WriteString("  filesystem → archive      add selected entries\\n\\n")\n\t\tbody.WriteString("Mouse: click selects,''',
    '''\t\tbody.WriteString("  filesystem → archive      add selected entries\\n\\n")\n\t\tbody.WriteString("F6 action:\\n")\n\t\tbody.WriteString("  filesystem → filesystem   move items or rename one item\\n\\n")\n\t\tbody.WriteString("Mouse: click selects,''',
)
