package main

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
