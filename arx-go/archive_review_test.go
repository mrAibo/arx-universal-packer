package main

import (
	"archive/tar"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

func TestCompatibilityCompressDoesNotOverwrite(t *testing.T) {
	base := t.TempDir()
	source := filepath.Join(base, "source.txt")
	if err := os.WriteFile(source, []byte("source"), 0o644); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(base, "target")
	if err := os.Mkdir(target, 0o755); err != nil {
		t.Fatal(err)
	}
	archive := filepath.Join(target, "bundle.tar.gz")
	if err := os.WriteFile(archive, []byte("keep-me"), 0o644); err != nil {
		t.Fatal(err)
	}

	result := compress("tar.gz", "bundle", source, target, 3)
	if result.Err == nil {
		t.Fatal("expected existing archive to be rejected")
	}
	content, err := os.ReadFile(archive)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "keep-me" {
		t.Fatalf("existing archive changed: %q", content)
	}
}

func TestCompatibilityCompressPropagatesTarFailure(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("test uses a POSIX shell script")
	}
	if _, err := exec.LookPath("gzip"); err != nil {
		t.Skip("gzip is not installed")
	}

	fakeBin := t.TempDir()
	fakeTar := filepath.Join(fakeBin, "tar")
	if err := os.WriteFile(fakeTar, []byte("#!/bin/sh\nprintf partial-output\nexit 42\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", fakeBin+string(os.PathListSeparator)+os.Getenv("PATH"))

	base := t.TempDir()
	source := filepath.Join(base, "source.txt")
	if err := os.WriteFile(source, []byte("source"), 0o644); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(base, "target")
	if err := os.Mkdir(target, 0o755); err != nil {
		t.Fatal(err)
	}

	result := compress("tar.gz", "broken", source, target, 3)
	if result.Err == nil {
		t.Fatal("expected tar failure to be returned")
	}
	if _, err := os.Stat(filepath.Join(target, "broken.tar.gz")); !os.IsNotExist(err) {
		t.Fatalf("partial archive was not removed: %v", err)
	}
}

func TestCompatibilityExtractRejectsTraversal(t *testing.T) {
	base := t.TempDir()
	archive := filepath.Join(base, "malicious.tar")
	file, err := os.Create(archive)
	if err != nil {
		t.Fatal(err)
	}
	writer := tar.NewWriter(file)
	payload := []byte("escape")
	if err := writer.WriteHeader(&tar.Header{Name: "../escape.txt", Mode: 0o644, Size: int64(len(payload))}); err != nil {
		t.Fatal(err)
	}
	if _, err := writer.Write(payload); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}

	target := filepath.Join(base, "output")
	result := extract(archive, target)
	if result.Err == nil {
		t.Fatal("expected unsafe archive member to be rejected")
	}
	if _, err := os.Stat(filepath.Join(base, "escape.txt")); !os.IsNotExist(err) {
		t.Fatalf("archive escaped extraction directory: %v", err)
	}
}

func TestCompatibilityConvertPreservesArchiveRoot(t *testing.T) {
	base := t.TempDir()
	sourceDir := filepath.Join(base, "docs")
	if err := os.Mkdir(sourceDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sourceDir, "readme.txt"), []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	created := compress("tar.gz", "source", sourceDir, base, 3)
	if created.Err != nil {
		t.Fatal(created.Err)
	}

	destination := filepath.Join(base, "converted.zip")
	converted := convert(filepath.Join(base, "source.tar.gz"), destination)
	if converted.Err != nil {
		t.Fatal(converted.Err)
	}
	items, err := readArchiveItems(destination)
	if err != nil {
		t.Fatal(err)
	}
	if !containsString(items, "docs/readme.txt") {
		t.Fatalf("converted archive lost its original root: %v", items)
	}
}
