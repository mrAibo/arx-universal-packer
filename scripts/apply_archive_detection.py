from pathlib import Path


def replace_exact(path: str, old: str, new: str, count: int = 1) -> None:
    file = Path(path)
    text = file.read_text()
    actual = text.count(old)
    if actual != count:
        raise SystemExit(f"{path}: expected {count} occurrence(s), found {actual}: {old!r}")
    file.write_text(text.replace(old, new))


replace_exact(
    "arx-go/archive.go",
    "// DetectFormat infers the archive format from the file extension.",
    "// DetectFormat infers an output format from the file extension.",
)
replace_exact(
    "arx-go/archive.go",
    "\tformat := DetectFormat(src)",
    "\tformat := DetectArchiveFormat(src)",
    count=2,
)

replace_exact(
    "arx-go/panel.go",
    '''\t\tinfo, infoErr := item.Info()\n\t\tentry := fileEntry{\n\t\t\tName:      item.Name(),\n\t\t\tPath:      filepath.Join(p.path, item.Name()),\n\t\t\tIsDir:     item.IsDir(),\n\t\t\tIsArchive: !item.IsDir() && DetectFormat(strings.ToLower(item.Name())) != "unknown",\n\t\t\tSize:      -1,\n\t\t}\n\t\tif infoErr == nil {\n\t\t\tentry.Size = info.Size()\n\t\t\tentry.ModTime = info.ModTime()\n\t\t}\n''',
    '''\t\tinfo, infoErr := item.Info()\n\t\tentry := fileEntry{\n\t\t\tName:  item.Name(),\n\t\t\tPath:  filepath.Join(p.path, item.Name()),\n\t\t\tIsDir: item.IsDir(),\n\t\t\tSize:  -1,\n\t\t}\n\t\tif infoErr == nil {\n\t\t\tentry.Size = info.Size()\n\t\t\tentry.ModTime = info.ModTime()\n\t\t\tentry.IsArchive = info.Mode().IsRegular() && DetectArchiveFormat(entry.Path) != "unknown"\n\t\t}\n''',
)
replace_exact(
    "arx-go/panel.go",
    "\tformat := DetectFormat(strings.ToLower(path))",
    "\tformat := DetectArchiveFormat(path)",
)

replace_exact(
    "arx-go/archive_selection.go",
    "\tformat := DetectFormat(strings.ToLower(source))",
    "\tformat := DetectArchiveFormat(source)",
)
replace_exact(
    "arx-go/archive_selection.go",
    "\tformat := DetectFormat(strings.ToLower(path))",
    "\tformat := DetectArchiveFormat(path)",
)
replace_exact(
    "arx-go/archive_edit.go",
    "\tformat := DetectFormat(strings.ToLower(archive))",
    "\tformat := DetectArchiveFormat(archive)",
)

replace_exact(
    "README.md",
    "- **Unified Interface**: One command for all formats (`tar`, `gz`, `bz2`, `xz`, `zst`, `zip`, `7z`).\n",
    "- **Unified Interface**: One command for all formats (`tar`, `gz`, `bz2`, `xz`, `zst`, `zip`, `7z`).\n- **Automatic Detection**: Existing archives are recognized by file signature, even when their extension is missing or misleading.\n",
)

Path("arx-go/archive_detection.go").write_text(r'''package main

import (
	"bytes"
	"compress/bzip2"
	"compress/gzip"
	"io"
	"os"
	"os/exec"
	"strconv"
)

var emptyTarBlock [512]byte

// DetectArchiveFormat identifies an existing archive by content first and
// falls back to its extension for corrupt or temporarily inaccessible files.
func DetectArchiveFormat(path string) string {
	if format := detectArchiveMagic(path); format != "unknown" {
		return format
	}
	return DetectFormat(path)
}

func detectArchiveMagic(path string) string {
	file, err := os.Open(path)
	if err != nil {
		return "unknown"
	}
	defer file.Close()

	header := make([]byte, 512)
	read, _ := io.ReadFull(file, header)
	header = header[:read]

	switch {
	case hasSignature(header, []byte{'P', 'K', 0x03, 0x04}),
		hasSignature(header, []byte{'P', 'K', 0x05, 0x06}),
		hasSignature(header, []byte{'P', 'K', 0x07, 0x08}):
		return "zip"
	case hasSignature(header, []byte{0x37, 0x7a, 0xbc, 0xaf, 0x27, 0x1c}):
		return "7z"
	case looksLikeTarHeader(header):
		return "tar"
	case hasSignature(header, []byte{0x1f, 0x8b}):
		if _, err := file.Seek(0, io.SeekStart); err != nil {
			return "unknown"
		}
		reader, err := gzip.NewReader(file)
		if err != nil {
			return "unknown"
		}
		isTar := readerHasTarHeader(reader)
		_ = reader.Close()
		if isTar {
			return "tar.gz"
		}
	case hasSignature(header, []byte{'B', 'Z', 'h'}):
		if _, err := file.Seek(0, io.SeekStart); err == nil && readerHasTarHeader(bzip2.NewReader(file)) {
			return "tar.bz2"
		}
	case hasSignature(header, []byte{0xfd, 0x37, 0x7a, 0x58, 0x5a, 0x00}):
		if commandHasTarHeader("xz", "-dc", "--", path) {
			return "tar.xz"
		}
	case hasSignature(header, []byte{0x28, 0xb5, 0x2f, 0xfd}):
		if commandHasTarHeader("zstd", "-qdc", "--", path) {
			return "tar.zst"
		}
	}
	return "unknown"
}

func hasSignature(data, signature []byte) bool {
	return len(data) >= len(signature) && bytes.Equal(data[:len(signature)], signature)
}

func readerHasTarHeader(reader io.Reader) bool {
	header := make([]byte, 512)
	_, err := io.ReadFull(reader, header)
	return err == nil && looksLikeTarHeader(header)
}

func commandHasTarHeader(name string, args ...string) bool {
	if _, err := exec.LookPath(name); err != nil {
		return false
	}
	command := exec.Command(name, args...)
	stdout, err := command.StdoutPipe()
	if err != nil {
		return false
	}
	command.Stderr = io.Discard
	if err := command.Start(); err != nil {
		return false
	}

	header := make([]byte, 512)
	_, readErr := io.ReadFull(stdout, header)
	_ = stdout.Close()
	if command.Process != nil {
		_ = command.Process.Kill()
	}
	_ = command.Wait()
	return readErr == nil && looksLikeTarHeader(header)
}

func looksLikeTarHeader(header []byte) bool {
	if len(header) < 512 || bytes.Equal(header[:512], emptyTarBlock[:]) {
		return false
	}
	stored, ok := tarChecksum(header[148:156])
	if !ok {
		return false
	}
	calculated := 0
	for index, value := range header[:512] {
		if index >= 148 && index < 156 {
			calculated += int(' ')
		} else {
			calculated += int(value)
		}
	}
	return stored == calculated
}

func tarChecksum(field []byte) (int, bool) {
	field = bytes.Trim(field, " \x00")
	if len(field) == 0 {
		return 0, false
	}
	value, err := strconv.ParseInt(string(field), 8, 64)
	return int(value), err == nil
}
''')

Path("arx-go/archive_detection_test.go").write_text(r'''package main

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestDetectArchiveFormatUsesContentBeforeExtension(t *testing.T) {
	path := filepath.Join(t.TempDir(), "misleading.tar.gz")
	writeTestZip(t, path)
	if got := DetectArchiveFormat(path); got != "zip" {
		t.Fatalf("DetectArchiveFormat()=%q want zip", got)
	}
}

func TestDetectArchiveFormatRecognizesExtensionlessArchives(t *testing.T) {
	directory := t.TempDir()
	tarPath := filepath.Join(directory, "plain-tar")
	writeTestTar(t, tarPath)
	if got := DetectArchiveFormat(tarPath); got != "tar" {
		t.Fatalf("tar format=%q", got)
	}

	gzipPath := filepath.Join(directory, "gzip-tar")
	writeTestTarGzip(t, gzipPath)
	if got := DetectArchiveFormat(gzipPath); got != "tar.gz" {
		t.Fatalf("tar.gz format=%q", got)
	}

	zipPath := filepath.Join(directory, "zip-archive")
	writeTestZip(t, zipPath)
	if got := DetectArchiveFormat(zipPath); got != "zip" {
		t.Fatalf("zip format=%q", got)
	}
}

func TestDetectArchiveFormatRecognizesExternalCompressedTar(t *testing.T) {
	directory := t.TempDir()
	tarPath := filepath.Join(directory, "payload.tar")
	writeTestTar(t, tarPath)

	cases := []struct {
		name    string
		command string
		args    []string
		want    string
	}{
		{name: "bzip2", command: "bzip2", args: []string{"-c"}, want: "tar.bz2"},
		{name: "xz", command: "xz", args: []string{"-c"}, want: "tar.xz"},
		{name: "zstd", command: "zstd", args: []string{"-q", "-c"}, want: "tar.zst"},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			if _, err := exec.LookPath(test.command); err != nil {
				t.Skipf("%s is not installed", test.command)
			}
			path := filepath.Join(directory, test.name+"-archive")
			output, err := os.Create(path)
			if err != nil {
				t.Fatal(err)
			}
			command := exec.Command(test.command, append(test.args, tarPath)...)
			command.Stdout = output
			command.Stderr = os.Stderr
			runErr := command.Run()
			closeErr := output.Close()
			if runErr != nil {
				t.Fatal(runErr)
			}
			if closeErr != nil {
				t.Fatal(closeErr)
			}
			if got := DetectArchiveFormat(path); got != test.want {
				t.Fatalf("format=%q want %q", got, test.want)
			}
		})
	}
}

func TestDetectArchiveFormatDoesNotTreatPlainGzipAsTar(t *testing.T) {
	path := filepath.Join(t.TempDir(), "compressed-data")
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	writer := gzip.NewWriter(file)
	if _, err := writer.Write([]byte("not a tar archive")); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	if got := DetectArchiveFormat(path); got != "unknown" {
		t.Fatalf("format=%q want unknown", got)
	}
}

func TestPanelRecognizesExtensionlessArchive(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "backup")
	writeTestZip(t, path)
	panel := newPane(directory)
	for _, entry := range panel.entries {
		if entry.Name == "backup" {
			if !entry.IsArchive {
				t.Fatal("extensionless ZIP was not marked as an archive")
			}
			return
		}
	}
	t.Fatal("backup entry not found")
}

func writeTestZip(t *testing.T, path string) {
	t.Helper()
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	writer := zip.NewWriter(file)
	entry, err := writer.Create("hello.txt")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.WriteString(entry, "hello"); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
}

func writeTestTar(t *testing.T, path string) {
	t.Helper()
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	writeTestTarStream(t, file)
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
}

func writeTestTarGzip(t *testing.T, path string) {
	t.Helper()
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	compressor := gzip.NewWriter(file)
	writeTestTarStream(t, compressor)
	if err := compressor.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
}

func writeTestTarStream(t *testing.T, output io.Writer) {
	t.Helper()
	writer := tar.NewWriter(output)
	content := []byte("hello")
	if err := writer.WriteHeader(&tar.Header{Name: "hello.txt", Mode: 0o644, Size: int64(len(content))}); err != nil {
		t.Fatal(err)
	}
	if _, err := writer.Write(content); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
}
''')
