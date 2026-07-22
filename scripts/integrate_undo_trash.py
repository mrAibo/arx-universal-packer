from pathlib import Path
import re


def patch(path: str, pattern: str, replacement: str) -> None:
    file = Path(path)
    text = file.read_text()
    updated, count = re.subn(pattern, replacement, text, count=1, flags=re.S)
    if count != 1:
        raise SystemExit(f"expected one match in {path}, got {count}: {pattern}")
    file.write_text(updated)


archive = Path("arx-go/archive_edit.go")
text = archive.read_text()
if "confirmFilesystemRestore" not in text:
    text = text.replace("\tconfirmFilesystemTrash\n", "\tconfirmFilesystemTrash\n\tconfirmFilesystemRestore\n", 1)
archive.write_text(text)

main = Path("arx-go/main.go")
text = main.read_text()
if 'case "ctrl+z"' not in text:
    text = text.replace(
        '\tcase "f8":\n\t\treturn m.startFilesystemTrash()\n',
        '\tcase "ctrl+z":\n\t\treturn m.startRestoreLastTrash()\n\tcase "f8":\n\t\treturn m.startFilesystemTrash()\n',
        1,
    )
    if 'case "ctrl+z"' not in text:
        raise SystemExit("F8 browser branch not found")
main.write_text(text)

confirm_file = next((p for p in Path("arx-go").glob("*.go") if "func (m model) updateConfirm" in p.read_text()), None)
if confirm_file is None:
    raise SystemExit("updateConfirm not found")
text = confirm_file.read_text()
if "case confirmFilesystemRestore:" not in text:
    text, count = re.subn(
        r'(\t\tcase confirmFilesystemTrash:\n\t\t\treturn m\.runFilesystemTrash\(entries\)\n)',
        r'\1\t\tcase confirmFilesystemRestore:\n\t\t\treturn m.runFilesystemRestore()\n',
        text,
        count=1,
    )
    if count != 1:
        raise SystemExit(f"trash confirmation branch not found in {confirm_file}")
confirm_file.write_text(text)

view = Path("arx-go/view.go")
text = view.read_text()
if "restore the last ARX trash operation" not in text:
    text, count = re.subn(
        r'(\t\tbody\.WriteString\("F8[^\n]*\\n"\)\n)',
        r'\1\t\tbody.WriteString("Ctrl-Z          restore the last ARX trash operation\\n")\n',
        text,
        count=1,
    )
    if count != 1:
        raise SystemExit("F8 help line not found")
view.write_text(text)

trash = Path("arx-go/filesystem_trash.go")
text = trash.read_text()
if "trashFilesystemPathRecord(entry.Path" not in text:
    text, count = re.subn(
        r'func trashFilesystem\(entries \[\]fileEntry\) Result \{.*?\n\}',
        '''func trashFilesystem(entries []fileEntry) Result {
\tfilesDir, infoDir, err := trashDirectories()
\tif err != nil {
\t\treturn Result{Err: err}
\t}
\tcompleted := 0
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
}''',
        text,
        count=1,
        flags=re.S,
    )
    if count != 1:
        raise SystemExit("trashFilesystem function not found")

if "func trashFilesystemPathRecord" not in text:
    replacement = '''func trashFilesystemPath(path, filesDir, infoDir string, deletedAt time.Time) error {
\t_, err := trashFilesystemPathRecord(path, filesDir, infoDir, deletedAt)
\treturn err
}

func trashFilesystemPathRecord(path, filesDir, infoDir string, deletedAt time.Time) (trashRecord, error) {
\tabsolute, err := filepath.Abs(path)
\tif err != nil {
\t\treturn trashRecord{}, err
\t}
\tif _, err := os.Lstat(absolute); err != nil {
\t\treturn trashRecord{}, err
\t}
\t_, dataPath, infoPath, err := availableTrashName(filepath.Base(absolute), filesDir, infoDir)
\tif err != nil {
\t\treturn trashRecord{}, err
\t}
\tinfoContent := fmt.Sprintf("[Trash Info]\\nPath=%s\\nDeletionDate=%s\\n", url.PathEscape(absolute), deletedAt.Format("2006-01-02T15:04:05"))
\ttemporary, err := os.CreateTemp(infoDir, ".arx-trashinfo-*")
\tif err != nil {
\t\treturn trashRecord{}, err
\t}
\ttemporaryName := temporary.Name()
\tdefer os.Remove(temporaryName)
\tif err := temporary.Chmod(0o600); err != nil {
\t\ttemporary.Close()
\t\treturn trashRecord{}, err
\t}
\tif _, err := temporary.WriteString(infoContent); err != nil {
\t\ttemporary.Close()
\t\treturn trashRecord{}, err
\t}
\tif err := temporary.Sync(); err != nil {
\t\ttemporary.Close()
\t\treturn trashRecord{}, err
\t}
\tif err := temporary.Close(); err != nil {
\t\treturn trashRecord{}, err
\t}
\tif err := moveFilesystemPath(absolute, dataPath, false); err != nil {
\t\treturn trashRecord{}, err
\t}
\tif err := os.Rename(temporaryName, infoPath); err != nil {
\t\trollbackErr := moveFilesystemPath(dataPath, absolute, false)
\t\tif rollbackErr != nil {
\t\t\treturn trashRecord{}, fmt.Errorf("write trash metadata: %w; rollback failed: %v", err, rollbackErr)
\t\t}
\t\treturn trashRecord{}, err
\t}
\treturn trashRecord{originalPath: absolute, dataPath: dataPath, infoPath: infoPath}, nil
}'''
    text, count = re.subn(
        r'func trashFilesystemPath\(path, filesDir, infoDir string, deletedAt time\.Time\) error \{.*?\n\}\n\n(?=func availableTrashName)',
        replacement + "\n\n",
        text,
        count=1,
        flags=re.S,
    )
    if count != 1:
        raise SystemExit("trashFilesystemPath function not found")
trash.write_text(text)
