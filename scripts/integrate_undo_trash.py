from pathlib import Path


def replace_once(path: Path, old: str, new: str) -> None:
    text = path.read_text()
    if old not in text:
        raise SystemExit(f"pattern not found in {path}: {old!r}")
    path.write_text(text.replace(old, new, 1))


archive_edit = Path("arx-go/archive_edit.go")
replace_once(
    archive_edit,
    "\tconfirmFilesystemTrash\n)",
    "\tconfirmFilesystemTrash\n\tconfirmFilesystemRestore\n)",
)

main = Path("arx-go/main.go")
replace_once(
    main,
    '\tcase "f8":\n\t\treturn m.startFilesystemTrash()\n',
    '\tcase "ctrl+z":\n\t\treturn m.startRestoreLastTrash()\n\tcase "f8":\n\t\treturn m.startFilesystemTrash()\n',
)

confirm_file = None
for candidate in Path("arx-go").glob("*.go"):
    if "func (m model) updateConfirm" in candidate.read_text():
        confirm_file = candidate
        break
if confirm_file is None:
    raise SystemExit("updateConfirm not found")
text = confirm_file.read_text()
needle = "\tcase confirmFilesystemTrash:\n\t\treturn m.runFilesystemTrash(entries)\n"
if needle not in text:
    raise SystemExit(f"trash confirmation branch not found in {confirm_file}")
text = text.replace(
    needle,
    needle + "\tcase confirmFilesystemRestore:\n\t\treturn m.runFilesystemRestore()\n",
    1,
)
confirm_file.write_text(text)

view = Path("arx-go/view.go")
replace_once(
    view,
    '\t\tbody.WriteString("F8              move filesystem items to trash; delete archive entries\\n")\n',
    '\t\tbody.WriteString("F8              move filesystem items to trash; delete archive entries\\n")\n\t\tbody.WriteString("Ctrl-Z          restore the last ARX trash operation\\n")\n',
)

trash = Path("arx-go/filesystem_trash.go")
text = trash.read_text()
old = '''\tcompleted := 0
\tfor _, entry := range entries {
\t\tif err := trashFilesystemPath(entry.Path, filesDir, infoDir, time.Now()); err != nil {
\t\t\treturn Result{Err: fmt.Errorf("trash %s: %w (%d of %d completed)", entry.Path, err, completed, len(entries))}
\t\t}
\t\tcompleted++
\t}
\treturn Result{Output: fmt.Sprintf("Moved %d item(s) to trash", completed)}
'''
new = '''\tcompleted := 0
\trecords := make([]trashRecord, 0, len(entries))
\tfor _, entry := range entries {
\t\trecord, err := trashFilesystemPathRecord(entry.Path, filesDir, infoDir, time.Now())
\t\tif err != nil {
\t\t\tsetLastTrashRecords(records)
\t\t\treturn Result{Err: fmt.Errorf("trash %s: %w (%d of %d completed)", entry.Path, err, completed, len(entries))}
\t\t}
\t\trecords = append(records, record)
\t\tcompleted++
\t}
\tsetLastTrashRecords(records)
\treturn Result{Output: fmt.Sprintf("Moved %d item(s) to trash", completed)}
'''
if old not in text:
    raise SystemExit("trash loop not found")
text = text.replace(old, new, 1)
old_sig = "func trashFilesystemPath(path, filesDir, infoDir string, deletedAt time.Time) error {"
new_sig = '''func trashFilesystemPath(path, filesDir, infoDir string, deletedAt time.Time) error {
\t_, err := trashFilesystemPathRecord(path, filesDir, infoDir, deletedAt)
\treturn err
}

func trashFilesystemPathRecord(path, filesDir, infoDir string, deletedAt time.Time) (trashRecord, error) {'''
if old_sig not in text:
    raise SystemExit("trash path function not found")
text = text.replace(old_sig, new_sig, 1)
start = text.index("func trashFilesystemPathRecord")
end = text.index("\nfunc availableTrashName", start)
block = text[start:end]
block = block.replace("\t\treturn err", "\t\treturn trashRecord{}, err")
block = block.replace("\t\treturn fmt.Errorf", "\t\treturn trashRecord{}, fmt.Errorf")
block = block.replace("\treturn nil\n}", "\treturn trashRecord{originalPath: absolute, dataPath: dataPath, infoPath: infoPath}, nil\n}", 1)
text = text[:start] + block + text[end:]
trash.write_text(text)
