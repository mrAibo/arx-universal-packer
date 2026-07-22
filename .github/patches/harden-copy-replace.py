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
    "arx-go/filesystem_copy.go",
    '''\tcase err == nil && !destinationInfo.IsDir():\n\t\tif err := os.RemoveAll(destination); err != nil {\n\t\t\treturn err\n\t\t}\n\t\tif err := os.Mkdir(destination, info.Mode().Perm()); err != nil {\n\t\t\treturn err\n\t\t}\n\tcase err == nil:''',
    '''\tcase err == nil && !destinationInfo.IsDir():\n\t\tstagingRoot, err := os.MkdirTemp(filepath.Dir(destination), ".arx-dir-*")\n\t\tif err != nil {\n\t\t\treturn err\n\t\t}\n\t\tdefer os.RemoveAll(stagingRoot)\n\t\tstaging := filepath.Join(stagingRoot, "replacement")\n\t\tif err := copyFilesystemDirectory(source, staging, info, false); err != nil {\n\t\t\treturn err\n\t\t}\n\t\treturn replaceFilesystemPath(staging, destination, true)\n\tcase err == nil:''',
)

replace_once(
    "arx-go/filesystem_copy.go",
    '''func replaceFilesystemPath(source, destination string, overwrite bool) error {\n\tif !overwrite {\n\t\treturn os.Rename(source, destination)\n\t}\n\tif err := os.Rename(source, destination); err == nil {\n\t\treturn nil\n\t}\n\tif err := os.RemoveAll(destination); err != nil {\n\t\treturn err\n\t}\n\treturn os.Rename(source, destination)\n}''',
    '''func replaceFilesystemPath(source, destination string, overwrite bool) error {\n\tsourceInfo, err := os.Lstat(source)\n\tif err != nil {\n\t\treturn err\n\t}\n\tdestinationInfo, err := os.Lstat(destination)\n\tif os.IsNotExist(err) {\n\t\treturn os.Rename(source, destination)\n\t}\n\tif err != nil {\n\t\treturn err\n\t}\n\tif !overwrite {\n\t\treturn os.ErrExist\n\t}\n\tif sourceInfo.IsDir() == destinationInfo.IsDir() && !destinationInfo.IsDir() {\n\t\treturn os.Rename(source, destination)\n\t}\n\n\tbackupFile, err := os.CreateTemp(filepath.Dir(destination), ".arx-backup-*")\n\tif err != nil {\n\t\treturn err\n\t}\n\tbackup := backupFile.Name()\n\tif err := backupFile.Close(); err != nil {\n\t\treturn err\n\t}\n\tif err := os.Remove(backup); err != nil {\n\t\treturn err\n\t}\n\tif err := os.Rename(destination, backup); err != nil {\n\t\treturn err\n\t}\n\tif err := os.Rename(source, destination); err != nil {\n\t\trollbackErr := os.Rename(backup, destination)\n\t\tif rollbackErr != nil {\n\t\t\treturn fmt.Errorf("replace failed: %w; rollback failed: %v", err, rollbackErr)\n\t\t}\n\t\treturn err\n\t}\n\treturn os.RemoveAll(backup)\n}''',
)

path = Path("arx-go/filesystem_copy_test.go")
text = path.read_text(encoding="utf-8")
marker = "func TestReplaceFilesystemPathKeepsDestinationWhenSourceIsMissing"
if marker in text:
    raise SystemExit("filesystem_copy_test.go: test already applied")
text = text.rstrip() + r'''

func TestReplaceFilesystemPathKeepsDestinationWhenSourceIsMissing(t *testing.T) {
	directory := t.TempDir()
	destination := filepath.Join(directory, "existing.txt")
	if err := os.WriteFile(destination, []byte("keep"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := replaceFilesystemPath(filepath.Join(directory, "missing.txt"), destination, true); err == nil {
		t.Fatal("expected missing source error")
	}
	content, err := os.ReadFile(destination)
	if err != nil || string(content) != "keep" {
		t.Fatalf("destination changed after failed replacement: %q err=%v", content, err)
	}
}
''' + "\n"
path.write_text(text, encoding="utf-8")
