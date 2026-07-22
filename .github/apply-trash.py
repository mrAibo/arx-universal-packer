from pathlib import Path


def replace_once(path: str, old: str, new: str) -> None:
    file = Path(path)
    text = file.read_text()
    count = text.count(old)
    if count != 1:
        raise SystemExit(f"{path}: expected one match, found {count}")
    file.write_text(text.replace(old, new, 1))


replace_once(
    "arx-go/archive_edit.go",
    "\tconfirmFilesystemMove\n)",
    "\tconfirmFilesystemMove\n\tconfirmFilesystemTrash\n)",
)
replace_once(
    "arx-go/archive_edit.go",
    "\t\tcase confirmFilesystemMove:\n\t\t\treturn m.runFilesystemMove(entries, destination, true)\n\t\tdefault:",
    "\t\tcase confirmFilesystemMove:\n\t\t\treturn m.runFilesystemMove(entries, destination, true)\n\t\tcase confirmFilesystemTrash:\n\t\t\treturn m.runFilesystemTrash(entries)\n\t\tdefault:",
)
replace_once(
    "arx-go/main.go",
    "\tcase \"f8\":\n\t\treturn m.startArchiveDelete()",
    "\tcase \"f8\":\n\t\treturn m.startFilesystemTrash()",
)
replace_once(
    "arx-go/view.go",
    "\treturn \"Clear\"\n}",
    "\treturn \"Trash\"\n}",
)
replace_once(
    "arx-go/view.go",
    "\t\tbody.WriteString(\"F8              delete archive entries; clear filesystem marks\\n\")",
    "\t\tbody.WriteString(\"F8              move filesystem items to trash; delete archive entries\\n\")",
)
