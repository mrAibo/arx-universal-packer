package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Result is what the backend returns to the UI.
type Result struct {
	Output string
	Err    error
}

func run(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// runIn runs a command with the working directory changed to dir.
func runIn(dir, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// runCapture runs a command and returns its combined output.
func runCapture(name string, args ...string) (string, error) {
	out, err := exec.Command(name, args...).CombinedOutput()
	return string(out), err
}

// DetectFormat infers an output format from the file extension.
func DetectFormat(path string) string {
	path = strings.ToLower(path)
	switch {
	case strings.HasSuffix(path, ".tar.gz"), strings.HasSuffix(path, ".tgz"):
		return "tar.gz"
	case strings.HasSuffix(path, ".tar.bz2"), strings.HasSuffix(path, ".tbz2"):
		return "tar.bz2"
	case strings.HasSuffix(path, ".tar.xz"), strings.HasSuffix(path, ".txz"):
		return "tar.xz"
	case strings.HasSuffix(path, ".tar.zst"):
		return "tar.zst"
	case strings.HasSuffix(path, ".tar"):
		return "tar"
	case strings.HasSuffix(path, ".zip"):
		return "zip"
	case strings.HasSuffix(path, ".7z"):
		return "7z"
	default:
		return "unknown"
	}
}

// compress is the compatibility entry point used by the older backend tests.
// Keep it on the same multi-source implementation as the TUI so overwrite,
// pipeline-error, and environment handling cannot drift between code paths.
func compress(format, name, source, targetDir string, level int) Result {
	src, err := filepath.Abs(source)
	if err != nil {
		return Result{Err: err}
	}
	if _, err := os.Stat(src); err != nil {
		return Result{Err: fmt.Errorf("file not found: %s", source)}
	}
	return compressMany(format, name, []string{src}, filepath.Dir(src), targetDir, level)
}

// extract delegates to the selective extractor, which validates archive member
// paths before invoking tar, unzip, or 7z.
func extract(path, targetDir string) Result {
	return extractSelected(path, nil, targetDir)
}

// list returns the normalized archive member list used by the TUI browser.
func list(path string) Result {
	items, err := readArchiveItems(path)
	if err != nil {
		return Result{Err: err}
	}
	return Result{Output: strings.Join(items, "\n")}
}

// convert reuses the modern conversion path so the extracted temporary
// directory itself is never introduced as an extra archive level.
func convert(src, dest string) Result {
	return convertArchive(src, dest, defaultLevel)
}
