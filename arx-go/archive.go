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

// extract validates and filters the archive member list before delegating to
// the selective extractor. Non-empty directory entries are omitted because
// tar extracts their descendants recursively; requesting both forms causes
// false "not found" failures. Empty directories remain explicit members.
func extract(path, targetDir string) Result {
	items, err := readArchiveItems(path)
	if err != nil {
		return Result{Err: err}
	}
	members := archiveExtractionMembers(items)
	if len(members) == 0 {
		return Result{Err: fmt.Errorf("archive has no safe extractable entries")}
	}
	return extractSelected(path, members, targetDir)
}

func archiveExtractionMembers(items []string) []string {
	type member struct {
		path  string
		isDir bool
	}

	normalized := make([]member, 0, len(items))
	for _, item := range items {
		path := normalizeArchivePath(item)
		if path == "" {
			continue
		}
		normalized = append(normalized, member{
			path:  path,
			isDir: strings.HasSuffix(strings.ReplaceAll(strings.TrimSpace(item), "\\", "/"), "/"),
		})
	}

	result := make([]string, 0, len(normalized))
	for index, candidate := range normalized {
		if candidate.isDir {
			prefix := candidate.path + "/"
			hasDescendant := false
			for otherIndex, other := range normalized {
				if otherIndex != index && strings.HasPrefix(other.path, prefix) {
					hasDescendant = true
					break
				}
			}
			if hasDescendant {
				continue
			}
		}
		result = append(result, candidate.path)
	}
	return safeUniqueMembers(result)
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
