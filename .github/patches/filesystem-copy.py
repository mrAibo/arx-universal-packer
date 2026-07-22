#!/usr/bin/env python3
from pathlib import Path


def replace_once(path: str, old: str, new: str) -> None:
    file = Path(path)
    text = file.read_text(encoding="utf-8")
    count = text.count(old)
    if count != 1:
        raise SystemExit(f"{path}: expected one replacement, found {count}")
    file.write_text(text.replace(old, new, 1), encoding="utf-8")


replace_once(
    "arx-go/main.go",
    '''\tconfirm        confirmKind\n\tconfirmArchive string\n\tconfirmEntries []fileEntry''',
    '''\tconfirm            confirmKind\n\tconfirmArchive     string\n\tconfirmEntries     []fileEntry\n\tconfirmDestination string''',
)

replace_once(
    "arx-go/main.go",
    '''\tcase "f2", "ctrl+a":\n\t\tactive.markAll()\n\t\tm.status = active.markSummary()''',
    '''\tcase "f2":\n\t\treturn m.startPack()\n\tcase "ctrl+a":\n\t\tactive.markAll()\n\t\tm.status = active.markSummary()''',
)

replace_once(
    "arx-go/main.go",
    '''\tm.confirm = confirmNone\n\tm.confirmArchive = ""\n\tm.confirmEntries = nil''',
    '''\tm.confirm = confirmNone\n\tm.confirmArchive = ""\n\tm.confirmEntries = nil\n\tm.confirmDestination = ""''',
)

replace_once(
    "arx-go/main.go",
    '''\tsources := make([]string, 0, len(entries))\n\tfor _, entry := range entries {\n\t\tsources = append(sources, entry.Path)\n\t}\n\tname := "archive"\n\tif len(entries) == 1 {\n\t\tname = defaultOutputName(entries[0].Path)\n\t}\n\treturn m.openArchiveDialog(actionPack, "Create archive", name, sources, active.path), nil\n}\n\nfunc (m model) startConvert()''',
    '''\treturn m.startFilesystemCopy(entries, passive.path)\n}\n\nfunc (m model) startPack() (tea.Model, tea.Cmd) {\n\tactive := m.panes[m.active]\n\tpassive := m.panes[1-m.active]\n\tif active.mode != paneFilesystem || passive.mode != paneFilesystem {\n\t\tm.showError(fmt.Errorf("archive creation requires filesystem panels"))\n\t\treturn m, nil\n\t}\n\tentries, err := active.operationEntries()\n\tif err != nil {\n\t\tm.showError(err)\n\t\treturn m, nil\n\t}\n\tsources := make([]string, 0, len(entries))\n\tfor _, entry := range entries {\n\t\tsources = append(sources, entry.Path)\n\t}\n\tname := "archive"\n\tif len(entries) == 1 {\n\t\tname = defaultOutputName(entries[0].Path)\n\t}\n\treturn m.openArchiveDialog(actionPack, "Create archive", name, sources, active.path), nil\n}\n\nfunc (m model) startConvert()''',
)

replace_once(
    "arx-go/archive_edit.go",
    '''const (\n\tconfirmNone confirmKind = iota\n\tconfirmArchiveDelete\n)''',
    '''const (\n\tconfirmNone confirmKind = iota\n\tconfirmArchiveDelete\n\tconfirmFilesystemCopy\n)''',
)

replace_once(
    "arx-go/archive_edit.go",
    '''func (m model) updateConfirm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {\n\tswitch msg.String() {\n\tcase "esc", "n", "q":\n\t\tm.closeModal()\n\t\treturn m, nil\n\tcase "enter", "y", "f8":\n\t\tif m.confirm != confirmArchiveDelete {\n\t\t\tm.closeModal()\n\t\t\treturn m, nil\n\t\t}\n\t\tarchive := m.confirmArchive\n\t\tentries := append([]fileEntry(nil), m.confirmEntries...)\n\t\tm.closeModal()\n\t\treturn m.startOperation(fmt.Sprintf("Deleting %d archive item(s)...", len(entries)), func() Result {\n\t\t\treturn deleteFromArchive(archive, entries, defaultLevel)\n\t\t})\n\tdefault:\n\t\treturn m, nil\n\t}\n}''',
    '''func (m model) updateConfirm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {\n\tswitch msg.String() {\n\tcase "esc", "n", "q":\n\t\tm.closeModal()\n\t\treturn m, nil\n\tcase "enter", "y", "f5", "f8":\n\t\tconfirmation := m.confirm\n\t\tarchive := m.confirmArchive\n\t\tentries := append([]fileEntry(nil), m.confirmEntries...)\n\t\tdestination := m.confirmDestination\n\t\tm.closeModal()\n\t\tswitch confirmation {\n\t\tcase confirmArchiveDelete:\n\t\t\treturn m.startOperation(fmt.Sprintf("Deleting %d archive item(s)...", len(entries)), func() Result {\n\t\t\t\treturn deleteFromArchive(archive, entries, defaultLevel)\n\t\t\t})\n\t\tcase confirmFilesystemCopy:\n\t\t\treturn m.runFilesystemCopy(entries, destination, true)\n\t\tdefault:\n\t\t\treturn m, nil\n\t\t}\n\tdefault:\n\t\treturn m, nil\n\t}\n}''',
)

replace_once(
    "arx-go/view.go",
    '''\t\t{"F1", "Help"},\n\t\t{"F2", "Mark"},''',
    '''\t\t{"F1", "Help"},\n\t\t{"F2", "Archive"},''',
)

replace_once(
    "arx-go/view.go",
    '''\treturn "Pack"\n}''',
    '''\treturn "Copy"\n}''',
)

replace_once(
    "arx-go/view.go",
    '''\t\tbody.WriteString("F2 / Ctrl-A     mark all visible items\\n")\n\t\tbody.WriteString("*                invert marks\\n")''',
    '''\t\tbody.WriteString("F2              create archive from selected filesystem items\\n")\n\t\tbody.WriteString("Ctrl-A           mark all visible items\\n")\n\t\tbody.WriteString("*                invert marks\\n")''',
)

replace_once(
    "arx-go/view.go",
    '''\t\tbody.WriteString("F5              pack/extract/add according to panel direction\\n")''',
    '''\t\tbody.WriteString("F5              copy/extract/add according to panel direction\\n")''',
)

replace_once(
    "arx-go/view.go",
    '''\t\tbody.WriteString("F5 direction:\\n")\n\t\tbody.WriteString("  filesystem → filesystem   create a new archive\\n")\n\t\tbody.WriteString("  archive → filesystem      extract selected entries\\n")\n\t\tbody.WriteString("  filesystem → archive      add selected entries\\n\\n")''',
    '''\t\tbody.WriteString("F2 action:\\n")\n\t\tbody.WriteString("  filesystem → filesystem   create a new archive\\n\\n")\n\t\tbody.WriteString("F5 direction:\\n")\n\t\tbody.WriteString("  filesystem → filesystem   copy selected entries\\n")\n\t\tbody.WriteString("  archive → filesystem      extract selected entries\\n")\n\t\tbody.WriteString("  filesystem → archive      add selected entries\\n\\n")''',
)

replace_once(
    "arx-go/navigation_test.go",
    '''\tif got := m.f5Label(); got != "Pack" {''',
    '''\tif got := m.f5Label(); got != "Copy" {''',
)
