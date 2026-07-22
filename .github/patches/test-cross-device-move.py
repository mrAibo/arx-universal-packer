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
    "arx-go/filesystem_move.go",
    '''func moveFilesystemPath(source, target string, overwrite bool) error {\n\terr := replaceFilesystemPath(source, target, overwrite)''',
    '''func moveFilesystemPath(source, target string, overwrite bool) error {\n\treturn moveFilesystemPathWithReplace(source, target, overwrite, replaceFilesystemPath)\n}\n\nfunc moveFilesystemPathWithReplace(source, target string, overwrite bool, replace func(string, string, bool) error) error {\n\terr := replace(source, target, overwrite)''',
)

replace_once(
    "arx-go/filesystem_move_test.go",
    '''\t"path/filepath"\n\t"testing"''',
    '''\t"path/filepath"\n\t"syscall"\n\t"testing"''',
)

path = Path("arx-go/filesystem_move_test.go")
text = path.read_text(encoding="utf-8")
marker = "func TestMoveFilesystemCrossDeviceFallback"
if marker in text:
    raise SystemExit("cross-device test already applied")
text = text.rstrip() + r'''

func TestMoveFilesystemCrossDeviceFallback(t *testing.T) {
	sourceRoot := t.TempDir()
	destination := t.TempDir()
	source := filepath.Join(sourceRoot, "large.dat")
	target := filepath.Join(destination, "large.dat")
	if err := os.WriteFile(source, []byte("cross-device"), 0o640); err != nil {
		t.Fatal(err)
	}
	initialReplace := func(string, string, bool) error {
		return syscall.EXDEV
	}
	if err := moveFilesystemPathWithReplace(source, target, false, initialReplace); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(source); !os.IsNotExist(err) {
		t.Fatalf("source still exists after committed fallback: %v", err)
	}
	content, err := os.ReadFile(target)
	if err != nil || string(content) != "cross-device" {
		t.Fatalf("fallback target=%q err=%v", content, err)
	}
}
''' + "\n"
path.write_text(text, encoding="utf-8")
